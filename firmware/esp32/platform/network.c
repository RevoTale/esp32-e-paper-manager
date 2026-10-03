#include "network.h"
#include "net_io.h"
#include "radio.h"
#include "provision.h"
#include "secure.h"
#include "esp_timer.h"
#include "esp_random.h"
#include "mbedtls/platform_util.h"
#include "freertos/task.h"
#include <stdatomic.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

typedef struct {
    ep_channel *channel;
    uint8_t config[512];
    uint64_t epoch, next, live;
    bool blocked;
} network;
static network net;
static portMUX_TYPE lock = portMUX_INITIALIZER_UNLOCKED;
static atomic_uint health;
static void state(unsigned value) {
    unsigned old = atomic_load(&health);
    while (!atomic_compare_exchange_weak(&health, &old, (old & ~255U) | value)) {}
}
static void failed(void) {
    unsigned old = atomic_load(&health), stage = old & 255U, count = (old >> 16) & 255U;
    if (count < 255) count++;
    atomic_store(&health, 6U | stage << 8 | count << 16);
}
void ep_network_health(uint8_t out[8]) {
    unsigned snapshot = atomic_load(&health);
    out[0] = 1; out[1] = (uint8_t)snapshot; out[2] = (uint8_t)(snapshot >> 8); out[3] = (uint8_t)(snapshot >> 16);
    uint64_t seconds = (uint64_t)esp_timer_get_time() / 1000000;
    ep_put_le(out + 4, seconds > UINT32_MAX ? UINT32_MAX : seconds, 4);
}
void ep_network_block(bool blocked) {
    taskENTER_CRITICAL(&lock);
    net.blocked = blocked;
    if (blocked) net.live = 0;
    taskEXIT_CRITICAL(&lock);
}
bool ep_network_live(uint64_t connection) {
    taskENTER_CRITICAL(&lock);
    bool live = !net.blocked && connection && net.live == connection;
    taskEXIT_CRITICAL(&lock);
    return live && ep_radio_online();
}
static bool publish(uint64_t connection) {
    taskENTER_CRITICAL(&lock);
    bool allowed = !net.blocked;
    if (allowed) net.live = connection;
    taskEXIT_CRITICAL(&lock);
    return allowed;
}
static bool blocked(void) {
    taskENTER_CRITICAL(&lock); bool value = net.blocked; taskEXIT_CRITICAL(&lock);
    return value;
}
static bool authenticate(int fd, uint64_t connection, ep_secure *session) {
    uint8_t preface[20] = {'E', 'P', 'N', '2'}, challenge[56], auth[72];
    memcpy(preface + 4, net.config + 24, 16);
    int64_t deadline = esp_timer_get_time() + 15000000;
    bool ok = ep_challenge(net.config + 40, net.config + 24, net.epoch, connection, challenge) &&
        ep_net_write(fd, preface, sizeof preface, deadline, connection) &&
        ep_net_write(fd, challenge, sizeof challenge, deadline, connection) &&
        ep_net_read(fd, auth, sizeof auth, deadline, connection) &&
        ep_secure_start(session, net.config + 40, net.config + 24, challenge, auth);
    mbedtls_platform_zeroize(auth, sizeof auth);
    return ok;
}
static bool exchange(int fd, uint64_t connection, ep_secure *session) {
    uint8_t envelope[EP_RECORD + 32]; ep_message request = {.connection = connection}, reply;
    // Idle authenticated sockets need no polling traffic or periodic frame replay.
    if (!ep_net_read(fd, envelope, 1, INT64_MAX, connection)) return false;
    int64_t deadline = esp_timer_get_time() + 5000000;
    if (!ep_net_read(fd, envelope + 1, 15, deadline, connection)) return false;
    size_t plain_size = (size_t)envelope[8] * 256 + envelope[9], received;
    if (!plain_size || plain_size > EP_RECORD ||
        !ep_net_read(fd, envelope + 16, plain_size + 16, deadline, connection) ||
        !ep_secure_open(session, envelope, plain_size + 32, request.bytes, sizeof request.bytes, &received)) return false;
    ep_record record;
    if (!ep_wire_decode(request.bytes, received, &record) || record.kind == 9) return false;
    request.size = (uint16_t)received;
    if (!ep_channel_exchange(net.channel, &request, &reply) || !ep_network_live(connection)) return false;
    size_t written;
    return ep_secure_seal(session, reply.bytes, reply.size, envelope, sizeof envelope, &written) &&
        ep_net_write(fd, envelope, written, esp_timer_get_time() + 5000000, connection);
}
static void run_connection(const char *host, const char *port, uint64_t connection) {
    state(3);
    int fd = ep_net_dial(host, port, connection);
    if (fd < 0) return;
    ep_secure session = {0}; state(4);
    if (authenticate(fd, connection, &session)) {
        state(5);
        while (exchange(fd, connection, &session)) {}
    }
    ep_secure_free(&session); close(fd);
    ep_channel_close(net.channel, connection);
}
static void run(void *context) {
    (void)context;
    if (!ep_radio_open(net.config)) { state(7); vTaskDelete(NULL); return; }
    char host[256] = {0}, port[6] = {0};
    size_t length = (size_t)net.config[20] * 256 + net.config[21];
    memcpy(host, net.config + 78, length - 6);
    char *separator = strrchr(host, ':');
    unsigned number = 0;
    for (const char *digit = separator + 1; *digit; digit++) number = number * 10 + (unsigned)(*digit - '0');
    // The validated numeric port may have many leading zeroes. Normalize it;
    // never copy its textual length into the six-byte service-name buffer.
    snprintf(port, sizeof port, "%u", number); *separator = 0;
    unsigned retries = 0;
    for (;;) {
        if (blocked()) { state(0); vTaskDelay(pdMS_TO_TICKS(100)); continue; }
        state(1);
        if (ep_radio_join()) {
            if (net.next == UINT64_MAX) { state(7); vTaskDelete(NULL); return; }
            uint64_t connection = ++net.next; // consumed before any challenge bytes
            int64_t started = esp_timer_get_time();
            if (!publish(connection)) continue;
            run_connection(host, port, connection);
            if (esp_timer_get_time() - started > 60000000) retries = 0;
        }
        if (blocked()) continue; // expected USB cancellation is not a failure
        failed();
        unsigned delay_ms = 1000U << (retries < 6 ? retries++ : 6);
        int64_t until = esp_timer_get_time() + (int64_t)(delay_ms + esp_random() % 501) * 1000;
        while (!blocked() && esp_timer_get_time() < until) vTaskDelay(pdMS_TO_TICKS(100));
    }
}
bool ep_network_start(ep_channel *channel, const uint8_t config[512], uint64_t epoch) {
    if (!epoch || !ep_config_valid(config)) { state(7); return false; }
    net.channel = channel; net.epoch = epoch; memcpy(net.config, config, sizeof net.config);
    if (xTaskCreate(run, "ep-network", 12288, NULL, 3, NULL) == pdPASS) return true;
    mbedtls_platform_zeroize(net.config, sizeof net.config);
    state(7); return false;
}
