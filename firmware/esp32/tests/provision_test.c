#include "provision.h"
#include <assert.h>
#include <string.h>

static uint8_t flash[16384];
static int read_data(void *ctx, size_t offset, void *dst, size_t n) {
    (void)ctx; assert(offset + n <= sizeof flash); memcpy(dst, flash + offset, n); return 0;
}
static int write_data(void *ctx, size_t offset, const void *src, size_t n) {
    (void)ctx; assert(offset + n <= sizeof flash); memcpy(flash + offset, src, n); return 0;
}
static int erase_data(void *ctx, size_t offset, size_t n) {
    (void)ctx; assert(offset + n <= 8192); memset(flash + offset, 255, n); return 0;
}
static void checksum(uint8_t *p) { ep_put_be32(p + 508, ep_crc32(p, 508)); }
int main(void) {
    ep_flash io = {NULL, read_data, write_data, erase_data};
    uint8_t request[512] = {0}, reply[512], config[512]; uint64_t generation;
    memset(flash, 255, sizeof flash);
    memcpy(request, "EPCQ", 4); request[4] = 2; request[5] = 1; checksum(request);
    assert(ep_provision(&io, request, reply) == 0);
    assert(!memcmp(reply, "EPCR", 4) && reply[6] == 0 && reply[12] == 0);
    request[5] = 2; request[16] = 2;
    request[17] = 4; request[18] = 8; request[19] = 11; request[21] = 20;
    memset(request + 24, 1, 48);
    memcpy(request + 72, "tcp://manager:12345", 18); request[21] = 18;
    memcpy(request + 327, "test", 4); memcpy(request + 359, "password", 8);
    memcpy(request + 422, "Europe/Kyiv", 11); checksum(request);
    assert(ep_provision_readonly(&io, request, reply) == 0 && reply[5] == 2 && reply[12] == 5);
    assert(ep_config_load(&io, config, &generation) == 0);
    request[508] ^= 1;
    assert(ep_provision_readonly(&io, request, reply) != 0);
    request[508] ^= 1;
    assert(ep_provision(&io, request, reply) == 0 && reply[6] == 1 && reply[12] == 0);
    assert(ep_config_load(&io, config, &generation) == 1 && generation == 1);
    assert(config[16] == 2 && !memcmp(config + 40, request + 40, 32));
    assert(ep_filled(reply + 391, 117, 0)); // no secret bytes in public response
    assert(ep_provision(&io, request, reply) == 0 && reply[12] != 0); // no overwrite
    request[5] = 3; request[40] = 3; checksum(request);
    assert(ep_provision(&io, request, reply) == 0 && reply[12] == 0);
    assert(ep_config_load(&io, config, &generation) == 1 && generation == 2);
    request[16] = 1; checksum(request); // explicit ESP32 WPA2, no reinterpretation
    assert(ep_provision(&io, request, reply) != 0);
    request[16] = 2; request[500] = 1; checksum(request);
    assert(ep_provision(&io, request, reply) != 0);
    memset(request + 6, 0, 502); request[5] = 4; checksum(request);
    assert(ep_provision(&io, request, reply) == 0 && reply[6] == 0);
    assert(ep_config_load(&io, config, &generation) == 0);
    for (size_t i = 8192; i < sizeof flash; i++) assert(flash[i] == 255);
}
