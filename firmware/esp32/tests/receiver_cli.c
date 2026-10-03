// Native peer for the REAL Go clients; physical GPIO/time/flash are test seams.
#include "panel75.h"
#include "provision.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

static uint8_t flash_bytes[16384], command;
static bool dc, region_mode, partial;
static uint64_t now = 180000000;
static unsigned refreshes, plane_bytes[2];
static int read_flash(void *c, size_t at, void *out, size_t n) {
    (void)c; assert(at <= sizeof flash_bytes && n <= sizeof flash_bytes - at);
    memcpy(out, flash_bytes + at, n); return 0;
}
static int write_flash(void *c, size_t at, const void *in, size_t n) {
    (void)c; assert(at <= sizeof flash_bytes && n <= sizeof flash_bytes - at);
    memcpy(flash_bytes + at, in, n); return 0;
}
static int erase_flash(void *c, size_t at, size_t n) {
    (void)c; assert(at <= sizeof flash_bytes && n <= sizeof flash_bytes - at);
    memset(flash_bytes + at, 255, n); return 0;
}
static int pin(void *c, ep_panel_pin p, bool high) {
    (void)c;
    if (p == EP_DC) dc = high;
    if (p == EP_RESET && !high) partial = false;
    return 0;
}
static int ready(void *c) { (void)c; return 1; }
static uint64_t clock_us(void *c) { (void)c; return now; }
static void delay(void *c, uint32_t us) { (void)c; now += us; }
static int spi(void *c, const uint8_t *data, size_t n) {
    (void)c; assert(n && n <= 64);
    if (!dc) { assert(n == 1); command = data[0]; if (command == 0x12) refreshes++; }
    else if (command == 0x50 && data[0] == 0xa9) partial = true;
    else if (command == 0x90) {
        const uint8_t window[9] = {0,240,0,255,0,254,0,255,1};
        assert(region_mode && partial && n == sizeof window && !memcmp(data, window, n));
    }
    else if (command == 0x10 || command == 0x13) {
        unsigned pass = command == 0x10 ? 0 : 1;
        uint8_t expected = refreshes ? 0x5a : 0xa5;
        if (!pass) expected = (uint8_t)~expected;
        if (partial) expected = pass ? 0xa5 : 0x5a;
        for (size_t i = 0; i < n; i++) assert(data[i] == expected);
        plane_bytes[pass] += (unsigned)n;
    }
    return 0;
}
int main(int argc, char **argv) {
    bool fast = argc == 2 && !strcmp(argv[1], "--region-fast");
    if (argc > 2 || (argc == 2 && strcmp(argv[1], "--region") && !fast)) return 1;
    region_mode = argc == 2;
    memset(flash_bytes, 255, sizeof flash_bytes);
    ep_flash flash = {NULL, read_flash, write_flash, erase_flash};
    ep_panel75 panel; ep_panel_io io = {NULL, spi, pin, ready, clock_us, delay};
    ep_screen screen; uint8_t boot[16] = {1}, id[16] = {2};
    // Test-only host scheduling budget: physical time is already a virtual seam.
    assert(ep_screen_init(&screen, (ep_screen_config){800,480,1000,1,2,1,fast ? 1u : 180000u}, ep_panel75_create(&panel, io), boot, id));
    if (region_mode) assert(ep_screen_regions(&screen, ep_panel75_regions()));
    uint8_t request[EP_RECORD], reply[EP_RECORD];
    for (;;) {
        size_t prefix = fread(request, 1, 4, stdin), size = 0;
        if (!prefix) break;
        if (prefix != 4) return 2;
        if (!memcmp(request, "EPCQ", 4)) {
            if (fread(request + 4, 1, 508, stdin) != 508 || ep_provision(&flash, request, reply)) return 3;
            size = 512;
        } else {
            if (fread(request + 4, 1, 28, stdin) != 28) return 4;
            size_t total = ep_wire_size(request);
            if (!total || fread(request + 32, 1, total - 32, stdin) != total - 32) return 5;
            ep_record record; if (!ep_wire_decode(request, total, &record)) return 6;
            uint64_t start = now / 1000;
            size = ep_screen_handle(&screen, &record, start, reply, sizeof reply);
            if (!size) return 7;
            uint8_t code = reply[33];
            ep_screen_completed(&screen, record.kind, code, start, now / 1000);
            size = ep_screen_reply(&screen, &record, code, reply, sizeof reply);
            if (record.kind == 6 && !code) now += 180000000; // virtual cooldown only
        }
        if (fwrite(reply, 1, size, stdout) != size || fflush(stdout)) return 8;
    }
    ep_screen_disconnect(&screen); mbedtls_sha256_free(&screen.hash);
    unsigned expected_bytes = region_mode ? 48004 : 96000;
    if (refreshes) assert(refreshes == 2 && plane_bytes[0] == expected_bytes && plane_bytes[1] == expected_bytes);
    return 0;
}
