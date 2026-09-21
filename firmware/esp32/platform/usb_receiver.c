#include "usb_receiver.h"
#include "uart.h"
#include "provision.h"
#include "esp_timer.h"
#include "esp_system.h"
#include "mbedtls/platform_util.h"
#include "freertos/task.h"
#include <string.h>

static bool read_record(ep_message *m) {
    int64_t deadline = esp_timer_get_time() + 5000000;
    if (!ep_uart_read(m->bytes + 1, 3, deadline)) return false;
    if (!memcmp(m->bytes, "EPCQ", 4)) m->size = 512;
    else if (!memcmp(m->bytes, "EPS2", 4)) {
        if (!ep_uart_read(m->bytes + 4, EP_HEADER - 4, deadline)) return false;
        size_t size = ep_wire_size(m->bytes);
        if (!size) return false;
        m->size = (uint16_t)size;
        if (!ep_uart_read(m->bytes + EP_HEADER, size - EP_HEADER, deadline)) return false;
        ep_record record;
        return ep_wire_decode(m->bytes, size, &record) && record.kind != 9;
    } else return false;
    return ep_uart_read(m->bytes + 4, 508, deadline) && ep_provision_valid(m->bytes);
}
static void receive(void *context) {
    ep_channel *channel = context;
    ep_message request = {.connection = 1}, reply;
    int64_t last = esp_timer_get_time();
    bool used = false;
    for (;;) {
        if (!ep_uart_poll_byte(request.bytes)) {
            if (used && esp_timer_get_time() - last >= 19000000) {
                ep_channel_close(channel, request.connection);
                if (request.connection == UINT64_MAX) { vTaskDelete(NULL); return; }
                request.connection++; used = false;
            }
            continue;
        }
        used = true;
        bool valid = read_record(&request);
        bool restart = false;
        if (valid) valid = ep_channel_exchange(channel, &request, &reply) &&
            ep_uart_write(reply.bytes, reply.size);
        if (valid && request.size == 512 && !memcmp(request.bytes, "EPCQ", 4))
            restart = request.bytes[5] >= 2 && request.bytes[5] <= 4 && reply.bytes[12] == 0;
        mbedtls_platform_zeroize(request.bytes, sizeof request.bytes);
        mbedtls_platform_zeroize(reply.bytes, sizeof reply.bytes);
        last = esp_timer_get_time();
        if (restart) esp_restart(); // Only after the authoritative EPCR was drained.
        if (valid) continue;
        ep_channel_close(channel, request.connection);
        if (request.connection == UINT64_MAX || ep_uart_discard() != ESP_OK) {
            vTaskDelete(NULL); return;
        }
        request.connection++;
    }
}
bool ep_usb_start(ep_channel *channel) {
    return xTaskCreate(receive, "ep-usb", 6144, channel, 4, NULL) == pdPASS;
}
