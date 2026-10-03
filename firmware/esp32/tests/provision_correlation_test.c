#include "provision.h"
#include <assert.h>
#include <string.h>

// USB-CORRELATION.md: v3 IDs are public correlation, never stored credentials.
static uint8_t flash[16384];
static unsigned writes;
static int read_data(void *ctx, size_t offset, void *dst, size_t n) {
    (void)ctx; assert(offset + n <= sizeof flash);
    memcpy(dst, flash + offset, n); return 0;
}
static int write_data(void *ctx, size_t offset, const void *src, size_t n) {
    (void)ctx; assert(offset + n <= sizeof flash); writes++;
    memcpy(flash + offset, src, n); return 0;
}
static int erase_data(void *ctx, size_t offset, size_t n) {
    (void)ctx; assert(offset + n <= 8192); writes++;
    memset(flash + offset, 255, n); return 0;
}
static ep_flash io = {NULL, read_data, write_data, erase_data};
static void checksum(uint8_t *p) { ep_put_be32(p + 508, ep_crc32(p, 508)); }
static void request_for(uint8_t *p, uint8_t operation) {
    memset(p, 0, 512); memcpy(p, "EPCQ", 4); p[4] = 3; p[5] = operation;
    for (unsigned i = 0; i < 16; i++) p[486 + i] = (uint8_t)(i + operation);
    if (operation == 2 || operation == 3) {
        p[16] = 2; p[17] = 4; p[18] = 8; p[19] = 11; p[21] = 18;
        memset(p + 24, 1, 16); memset(p + 40, 0xab, 32);
        memcpy(p + 72, "tcp://manager:12345", 18);
        memcpy(p + 327, "test", 4); memcpy(p + 359, "password", 8);
        memcpy(p + 422, "Europe/Kyiv", 11);
    }
    checksum(p);
}
static void assert_correlated(const uint8_t *request, const uint8_t *reply) {
    assert(memcmp(reply, "EPCR", 4) == 0);
    assert(reply[4] == 3 && reply[5] == request[5]);
    assert(memcmp(reply + 392, request + 486, 16) == 0);
    assert(reply[391] == 0 && ep_filled(reply + 408, 100, 0));
    assert(ep_be32(reply + 508) == ep_crc32(reply, 508));
}
static void inspect_echoes_id(void) {
    uint8_t request[512], reply[512];
    memset(flash, 255, sizeof flash); writes = 0; request_for(request, 1);
    assert(ep_provision(&io, request, reply) == 0);
    assert_correlated(request, reply);
    assert(reply[6] == 0 && reply[12] == 0 && writes == 0);
}
static void mutations_keep_public_metadata(void) {
    uint8_t request[512], reply[512], config[512]; uint64_t generation;
    memset(flash, 255, sizeof flash); request_for(request, 2);
    assert(ep_provision(&io, request, reply) == 0);
    assert_correlated(request, reply);
    assert(reply[6] == 1 && reply[12] == 0 && ep_be64(reply + 16) == 1);
    assert(reply[7] == 2 && reply[8] == 4 && reply[9] == 11 && reply[11] == 18);
    assert(memcmp(reply + 24, request + 24, 16) == 0);
    assert(memcmp(reply + 40, "test", 4) == 0 && ep_filled(reply + 44, 28, 0));
    assert(memcmp(reply + 72, "tcp://manager:12345", 18) == 0);
    assert(memcmp(reply + 327, "Europe/Kyiv", 11) == 0);
    assert(ep_config_load(&io, config, &generation) == 1);
    assert(config[4] == 1 && ep_filled(config + 486, 22, 0));
    request_for(request, 3); request[40] = 0xcd; checksum(request);
    assert(ep_provision(&io, request, reply) == 0);
    assert_correlated(request, reply);
    assert(reply[12] == 0 && ep_be64(reply + 16) == 2);
    request_for(request, 4);
    assert(ep_provision(&io, request, reply) == 0);
    assert_correlated(request, reply);
    assert(reply[6] == 0 && reply[12] == 0);
    assert(ep_config_load(&io, config, &generation) == 0);
    assert(ep_filled(flash + 8192, 8192, 255));
}
static void invalid_ids_and_padding_cannot_mutate(void) {
    uint8_t request[512], reply[512];
    memset(flash, 255, sizeof flash); writes = 0;
    for (uint8_t operation = 1; operation <= 5; operation++) {
        request_for(request, operation); memset(request + 486, 0, 16); checksum(request);
        assert(!ep_provision_valid(request));
        assert(ep_provision(&io, request, reply) != 0);
        for (size_t offset = 502; offset < 508; offset++) {
            request_for(request, operation); request[offset] = 1; checksum(request);
            assert(!ep_provision_valid(request));
            assert(ep_provision_readonly(&io, request, reply) != 0);
        }
    }
    assert(writes == 0 && ep_filled(flash, sizeof flash, 255));
}
static void readonly_recovery_echoes_id(void) {
    uint8_t request[512], reply[512];
    memset(flash, 255, sizeof flash); writes = 0;
    for (uint8_t operation = 1; operation <= 5; operation++) {
        request_for(request, operation);
        assert(ep_provision_readonly(&io, request, reply) == 0);
        assert_correlated(request, reply);
        assert(reply[12] == 5 && reply[6] == 0);
        assert(ep_provision_readonly(NULL, request, reply) == 0);
        assert_correlated(request, reply);
        assert(reply[12] == 3 && reply[6] == 3);
    }
    assert(writes == 0);
}
static void legacy_v2_stays_uncorrelated(void) {
    uint8_t request[512], reply[512];
    memset(flash, 255, sizeof flash);
    request_for(request, 1); request[4] = 2; memset(request + 486, 0, 16); checksum(request);
    assert(ep_provision(&io, request, reply) == 0);
    assert(reply[4] == 2 && reply[5] == 1 && reply[12] == 0);
    assert(ep_filled(reply + 391, 117, 0));
    assert(ep_be32(reply + 508) == ep_crc32(reply, 508));
    request[486] = 1; checksum(request);
    assert(!ep_provision_valid(request));
}
int main(void) {
    legacy_v2_stays_uncorrelated();
    inspect_echoes_id();
    mutations_keep_public_metadata();
    invalid_ids_and_padding_cannot_mutate();
    readonly_recovery_echoes_id();
}
