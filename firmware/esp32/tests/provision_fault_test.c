#include "provision.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

static uint8_t storage[16384];
static unsigned reads, writes, erases, fail_read, erase_prefix;
static bool write_error, erase_noop, lose_verified;
static int read_flash(void *context, size_t offset, void *out, size_t size) {
    (void)context;
    assert(offset <= sizeof storage && size <= sizeof storage - offset);
    if (++reads == fail_read) return -1;
    memcpy(out, storage + offset, size);
    if (lose_verified && reads == 3) memset(storage + offset, 255, 4096);
    return 0;
}
static int write_flash(void *context, size_t offset, const void *in, size_t size) {
    (void)context;
    assert((offset == 0 || offset == 4096) && size == 512);
    writes++;
    const uint8_t *source = in;
    for (size_t i = 0; i < size; i++) storage[offset + i] &= source[i];
    return write_error ? -1 : 0;
}
static int erase_flash(void *context, size_t offset, size_t size) {
    (void)context;
    assert(offset <= 8192 && size <= 8192 - offset);
    erases++;
    if (erase_noop) return 0;
    if (erase_prefix) {
        assert(erase_prefix < size);
        memset(storage + offset, 255, erase_prefix);
        return -1;
    }
    memset(storage + offset, 255, size);
    return 0;
}
static const ep_flash io = {NULL, read_flash, write_flash, erase_flash};
static void faults_clear(void) {
    reads = writes = erases = fail_read = erase_prefix = 0;
    write_error = erase_noop = lose_verified = false;
}
static void fresh(void) {
    memset(storage, 255, sizeof storage);
    faults_clear();
}
static void checksum(uint8_t *p) { ep_put_be32(p + 508, ep_crc32(p, 508)); }
static void request_make(uint8_t *p, uint8_t operation) {
    memset(p, 0, 512);
    memcpy(p, "EPCQ", 4); p[4] = 2; p[5] = operation;
    if (operation == 2 || operation == 3) {
        p[16] = 2; p[17] = 4; p[18] = 8; p[19] = 11; p[21] = 18;
        memset(p + 24, 0x35, 16); memset(p + 40, 0xa6, 32);
        memcpy(p + 72, "tcp://manager:12345", 18);
        memcpy(p + 327, "test", 4); memcpy(p + 359, "password", 8);
        memcpy(p + 422, "Europe/Kyiv", 11);
    }
    checksum(p);
}
static void reply_check(const uint8_t *p, unsigned state, unsigned code) {
    assert(!memcmp(p, "EPCR", 4) && p[4] == 2);
    if (p[6] != state || p[12] != code)
        fprintf(stderr, "operation %u: state/code %u/%u, expected %u/%u\n", p[5], p[6], p[12], state, code);
    assert(p[6] == state && p[12] == code);
    assert(ep_be32(p + 508) == ep_crc32(p, 508));
    assert(ep_filled(p + 13, 3, 0) && ep_filled(p + 391, 117, 0));
    if (state != 1) assert(ep_filled(p + 16, 375, 0));
    for (unsigned i = 0; i + 8 <= 508; i++) {
        assert(memcmp(p + i, "password", 8));
        assert(!ep_filled(p + i, 8, 0xa6));
    }
}
static void rejected(uint8_t *request) {
    uint8_t reply[512];
    checksum(request);
    faults_clear();
    assert(ep_provision(&io, request, reply) == -1);
    assert(reads == 0 && writes == 0 && erases == 0);
}
static void parser_bounds(void) {
    uint8_t request[512];
    const unsigned offsets[] = {17, 18, 19, 20, 21};
    const unsigned invalid[] = {33, 64, 65, 1, 0};
    for (unsigned i = 0; i < sizeof offsets / sizeof offsets[0]; i++) {
        request_make(request, 2); request[offsets[i]] = (uint8_t)invalid[i]; rejected(request);
    }
    const unsigned padding[] = {6, 15, 22, 23, 90, 326, 331, 358, 367, 421, 433, 485, 486, 507};
    for (unsigned i = 0; i < sizeof padding / sizeof padding[0]; i++) {
        request_make(request, 2); request[padding[i]] = 1; rejected(request);
    }
    for (unsigned i = 6; i < 508; i++) {
        request_make(request, 1); request[i] = 1; rejected(request);
    }
    const uint8_t invalid_utf8[][4] = {
        {0xc0, 0xaf, 'x', 'x'}, {0xed, 0xa0, 0x80, 'x'}, {0xf4, 0x90, 0x80, 0x80},
        {'x', 'x', 'x', 0xc2}, {0x80, 'x', 'x', 'x'}, {0xe0, 0x80, 0x80, 'x'},
        {0, 'x', 'x', 'x'}, {127, 'x', 'x', 'x'}
    };
    for (unsigned i = 0; i < sizeof invalid_utf8 / sizeof invalid_utf8[0]; i++) {
        request_make(request, 2); memcpy(request + 327, invalid_utf8[i], 4); rejected(request);
    }
    request_make(request, 2); memcpy(request + 327, "\xf4\x8f\xbf\xbf", 4);
    assert(ep_config_valid(request)); // Highest Unicode scalar remains accepted.
    const char *bad_manager[] = {"tcp://a:0", "tcp://a:65536", "tcp://-a:1", "tcp://a..b:1", "tcp://a:1/x", "tcp://a:1?x", "tcp://u@a:1", "tcp://[::1]:1"};
    for (unsigned i = 0; i < sizeof bad_manager / sizeof bad_manager[0]; i++) {
        request_make(request, 2); memset(request + 72, 0, 255);
        size_t size = strlen(bad_manager[i]); memcpy(request + 72, bad_manager[i], size);
        request[21] = (uint8_t)size; rejected(request);
    }
}
static void operation_failures(void) {
    uint8_t request[512], reply[512], config[512]; uint64_t generation;
    fresh(); request_make(request, 2); write_error = true;
    assert(ep_provision(&io, request, reply) == 0); reply_check(reply, 1, 3);
    assert(ep_config_load(&io, config, &generation) == 1 && generation == 1);
    faults_clear(); request_make(request, 3); request[40] = 0x78; checksum(request);
    assert(ep_provision(&io, request, reply) == 0); reply_check(reply, 1, 0);
    faults_clear(); request_make(request, 4); erase_prefix = 4096;
    assert(ep_provision(&io, request, reply) == 0); reply_check(reply, 1, 3);
    assert(ep_be64(reply + 16) == 1); // Partial reset exposes surviving older slot.
    faults_clear(); fail_read = 2;
    memset(config, 0xa6, sizeof config); generation = 99;
    assert(ep_config_load(&io, config, &generation) == 3);
    assert(generation == 0 && ep_filled(config, sizeof config, 0));
    fresh(); request_make(request, 2); fail_read = 4;
    assert(ep_provision(&io, request, reply) == 0); reply_check(reply, 3, 3);
    for (size_t i = 8192; i < sizeof storage; i++) assert(storage[i] == 255);
}
static void postconditions(void) {
    uint8_t request[512], reply[512];
    fresh(); request_make(request, 2);
    assert(ep_provision(&io, request, reply) == 0); reply_check(reply, 1, 0);
    faults_clear(); request_make(request, 4); erase_noop = true;
    assert(ep_provision(&io, request, reply) == 0); reply_check(reply, 1, 3);
    fresh(); request_make(request, 2); lose_verified = true;
    assert(ep_provision(&io, request, reply) == 0); reply_check(reply, 0, 3);
    fresh(); request_make(request, 2);
    assert(ep_provision(&io, request, reply) == 0);
    faults_clear(); request_make(request, 3); request[40] = 0x78; checksum(request);
    lose_verified = true;
    assert(ep_provision(&io, request, reply) == 0); reply_check(reply, 1, 3);
    assert(ep_be64(reply + 16) == 1);
}
int main(void) {
    ep_flash unavailable = {0}; uint8_t request[512], reply[512];
    request_make(request, 1);
    assert(ep_provision_readonly(&unavailable, request, reply) == 0);
    reply_check(reply, 3, 3);
    fresh(); parser_bounds(); operation_failures(); postconditions();
    puts("provision faults: parser bounds, secret-free responses, I/O and postconditions passed");
}
