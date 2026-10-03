#include "screen.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

static unsigned begins, writes, commits, aborts;
static uint8_t digest[32], pixels[2] = {0};
static int begin(void *context) { (void)context; begins++; return 0; }
static int write_pixels(void *context, uint8_t pass, uint32_t offset, const uint8_t *bytes, size_t size) {
    (void)context; assert(pass == 0 && offset + size <= sizeof pixels && bytes); writes++; return 0;
}
static int commit(void *context) { (void)context; commits++; return 0; }
static int abort_pixels(void *context) { (void)context; aborts++; return 0; }
static uint8_t send(ep_screen *screen, ep_record request, uint64_t now) {
    uint8_t bytes[120]; ep_record reply;
    size_t size = ep_screen_handle(screen, &request, now, bytes, sizeof bytes);
    assert(ep_wire_decode(bytes, size, &reply));
    return reply.payload[1];
}
static ep_record transaction(uint8_t kind, uint64_t id) {
    return (ep_record){.kind=kind, .epoch=1, .id=id, .size=32, .payload=digest};
}
static ep_record data(uint32_t offset) {
    return (ep_record){.kind=5, .epoch=1, .id=1, .offset=offset, .size=1, .payload=pixels};
}
static void fresh(ep_screen *screen) {
    begins = writes = commits = aborts = 0;
    uint8_t boot[16] = {1}, identity[16] = {2}, claim[32] = {1};
    memset(claim + 16, 3, 16);
    ep_sink sink = {NULL, begin, write_pixels, commit, abort_pixels};
    assert(ep_screen_init(screen, (ep_screen_config){8, 2, 2, 1, 1, 42, 1000}, sink, boot, identity));
    assert(mbedtls_sha256(pixels, sizeof pixels, digest, 0) == 0);
    assert(send(screen, (ep_record){.kind=2, .payload=claim, .size=32}, 0) == 0);
    assert(send(screen, (ep_record){.kind=3, .epoch=1, .payload=claim, .size=32}, 0) == 0);
}
static void blocking_begin_and_data(void) {
    ep_screen screen; fresh(&screen);
    assert(send(&screen, transaction(4, 1), 1000) == 0);
    ep_screen_completed(&screen, 4, 0, 1000, 31000);
    assert(screen.started == 31000 && screen.progress == 31000 && screen.observed == 31000);
    assert(send(&screen, data(0), 46000) == 0); // Only 15 seconds of host waiting.
    ep_screen_completed(&screen, 5, 0, 46000, 81000);
    assert(screen.started == 66000 && screen.progress == 81000 && screen.observed == 81000);
    assert(send(&screen, data(1), 82000) == 0 && screen.state == 2);
    ep_screen_completed(&screen, 5, 0, 82000, 107000);
    assert(screen.started == 91000 && screen.progress == 107000 && screen.observed == 107000);
    assert(send(&screen, transaction(6, 1), 107001) == 0 && commits == 1);
    ep_screen_completed(&screen, 6, 0, 107001, 137001);
    assert(screen.last_refresh == 137001 && screen.observed == 137001);
    assert(send(&screen, transaction(6, 1), 137500) == 0 && commits == 1);
    ep_screen_completed(&screen, 6, 0, 137500, 137800);
    assert(screen.last_refresh == 137001 && screen.observed == 137800);
    assert(send(&screen, transaction(4, 2), 138000) == 6);
    assert(send(&screen, transaction(4, 2), 138001) == 0);
    assert(begins == 2 && writes == 2 && commits == 1 && aborts == 0);
}
static void queries_do_not_extend_progress(void) {
    ep_screen screen; fresh(&screen);
    assert(send(&screen, transaction(4, 1), 1000) == 0);
    ep_screen_completed(&screen, 4, 0, 1000, 2000);
    assert(send(&screen, transaction(7, 1), 10000) == 0);
    ep_screen_completed(&screen, 7, 0, 10000, 12000);
    assert(screen.started == 2000 && screen.progress == 2000 && screen.observed == 12000);
    assert(send(&screen, (ep_record){.kind=1}, 13000) == 0);
    ep_screen_completed(&screen, 1, 0, 13000, 14000);
    assert(screen.started == 2000 && screen.progress == 2000 && screen.observed == 14000);
    assert(ep_screen_tick(&screen, 21999) == 0);
    assert(ep_screen_tick(&screen, 22000) == 10 && aborts == 1);
}
static void failures_and_regressions(void) {
    ep_screen screen;
    const uint8_t operations[] = {4, 5, 6};
    for (unsigned i = 0; i < sizeof operations; i++) {
        fresh(&screen); assert(send(&screen, transaction(4, 1), 1000) == 0);
        ep_screen_completed(&screen, operations[i], 3, 1000, 2000);
        assert(screen.started == 1000 && screen.progress == 1000 && screen.last_refresh == 0);
        assert(screen.observed == 2000);
    }
    fresh(&screen); assert(send(&screen, transaction(4, 1), 1000) == 0);
    ep_screen_completed(&screen, 4, 0, 1000, 999);
    assert(screen.state == 4 && screen.failure == 10 && aborts == 1);
    assert(screen.observed == 1000 && screen.progress == 1000);
    assert(send(&screen, transaction(4, 1), 1001) == 4); // Invalid timing cannot unconsume ID.
    fresh(&screen); assert(send(&screen, transaction(4, 1), 1000) == 0);
    ep_screen_completed(&screen, 4, 0, 1000, 2000);
    ep_screen_completed(&screen, 5, 0, 1500, 3000); // Stale completion cannot move a newer observation.
    assert(screen.state == 4 && screen.failure == 10 && aborts == 1 && screen.observed == 2000);
}
int main(void) {
    blocking_begin_and_data(); queries_do_not_extend_progress(); failures_and_regressions();
    puts("screen completion: blocking time, cadence, readonly progress and clock regressions passed");
}
