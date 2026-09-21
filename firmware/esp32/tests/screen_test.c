#include "screen.h"
#include <assert.h>
#include <string.h>

static unsigned begins, commits, aborts, writes;
static int begin(void *ctx) { (void)ctx; begins++; return 0; }
static int write_pixels(void *ctx, uint8_t pass, uint32_t offset, const uint8_t *data, size_t n) {
    (void)ctx; assert(pass < 2 && offset + n <= 48000 && data); writes++; return 0;
}
static int commit(void *ctx) { (void)ctx; commits++; return 0; }
static int abort_pixels(void *ctx) { (void)ctx; aborts++; return 0; }
static uint8_t send(ep_screen *screen, ep_record record, uint64_t now) {
    uint8_t reply[120]; ep_record decoded;
    size_t n = ep_screen_handle(screen, &record, now, reply, sizeof reply);
    assert(ep_wire_decode(reply, n, &decoded)); assert(decoded.kind == 9);
    return decoded.payload[1];
}
int main(void) {
    ep_screen screen;
    ep_sink sink = {NULL, begin, write_pixels, commit, abort_pixels};
    uint8_t boot[16] = {1}, id[16] = {2}, claim[32] = {0}, pixels[48000] = {0}, digest[32];
    memcpy(claim, boot, 16); memset(claim + 16, 5, 16);
    assert(mbedtls_sha256(pixels, sizeof pixels, digest, 0) == 0);
    ep_screen_config profile = {800, 480, 1000, 1, 2, 1, 180000};
    assert(ep_screen_init(&screen, profile, sink, boot, id));
    assert(send(&screen, (ep_record){.kind=1}, 0) == 0);
    assert(send(&screen, (ep_record){.kind=2, .payload=claim, .size=32}, 0) == 0);
    assert(screen.generation == 1);
    assert(send(&screen, (ep_record){.kind=2, .payload=claim, .size=32}, 0) == 0);
    assert(screen.generation == 1); // lost acquire ACK does not increment twice
    assert(send(&screen, (ep_record){.kind=3, .epoch=1, .payload=claim, .size=32}, 0) == 0);
    ep_record tx = {.kind=4, .epoch=1, .id=1, .size=32, .payload=digest};
    assert(send(&screen, tx, 0) == 6 && begins == 0); // conservative reboot floor
    assert(send(&screen, tx, 180000) == 0 && begins == 1);
    for (uint8_t pass = 0; pass < 2; pass++) {
        for (uint32_t offset = 0; offset < 48000; offset += 1000) {
            ep_record data = {.kind=5, .epoch=1, .id=1, .pass=pass, .offset=offset, .size=1000, .payload=pixels+offset};
            assert(send(&screen, data, 180001) == 0);
        }
    }
    assert(screen.state == 2 && writes == 96 && commits == 0);
    tx.kind = 6;
    assert(send(&screen, tx, 180002) == 0 && commits == 1 && screen.current_image);
    assert(send(&screen, tx, 180003) == 0 && commits == 1);
    ep_screen_disconnect(&screen);
    assert(aborts == 0);
    assert(send(&screen, (ep_record){.kind=3, .epoch=1, .payload=claim, .size=32}, 180004) == 0);
    tx.kind = 7; assert(send(&screen, tx, 180004) == 0 && screen.current_image);
    tx.kind = 4; assert(send(&screen, tx, 360005) == 4); // ID consumed forever in lease
    tx.id = 2; assert(send(&screen, tx, 360005) == 0);
    ep_screen_tick(&screen, 380006);
    assert(screen.state == 4 && aborts == 1 && commits == 1);

    // A configured urgent budget bypasses normal cadence, never active staging,
    // unsupported modes, consumed identity, or a fatal panel failure.
    uint8_t options[44] = {0}; memcpy(options, digest, 32);
    options[32] = 1; options[33] = 2;
    ep_put_le(options + 36, 180000, 4); ep_put_le(options + 40, 30000, 4);
    tx = (ep_record){.kind=13, .epoch=1, .id=3, .size=44, .payload=options};
    screen.last_refresh = 380006;
    assert(send(&screen, tx, 400006) == 6 && begins == 2);
    options[33] = 1; // partial is not implemented by this adapter
    assert(send(&screen, tx, 410006) == 12 && begins == 2);
    options[33] = 2; options[34] = 1;
    assert(send(&screen, tx, 410006) == 12 && begins == 2);
    options[34] = 0;
    assert(send(&screen, tx, 410006) == 0 && begins == 3);
    tx.id = 4;
    assert(send(&screen, tx, 410007) == 3 && begins == 3);
    assert(send(&screen, (ep_record){.kind=8, .epoch=1}, 410008) == 0);
    tx.id = 3;
    assert(send(&screen, tx, 410009) == 4 && begins == 3);
    screen.fatal = true; tx.id = 4;
    assert(send(&screen, tx, 410010) == 11 && begins == 3);
}
