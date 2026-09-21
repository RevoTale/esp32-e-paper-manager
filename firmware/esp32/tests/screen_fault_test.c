#include "screen.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

static unsigned begins, writes, commits, aborts;
static bool fail_begin, fail_write, fail_commit, fail_abort;
static uint8_t boot[16] = {1}, identity[16] = {2}, claim[32], pixels[48000], digest[32];
static int begin(void *context) { (void)context; begins++; return fail_begin ? -1 : 0; }
static int write_pixels(void *context, uint8_t pass, uint32_t offset, const uint8_t *data, size_t size) {
    (void)context;
    assert(pass < 2 && offset <= sizeof pixels && size <= sizeof pixels - offset && data);
    writes++; return fail_write ? -1 : 0;
}
static int commit(void *context) { (void)context; commits++; return fail_commit ? -1 : 0; }
static int abort_pixels(void *context) { (void)context; aborts++; return fail_abort ? -1 : 0; }
static uint8_t response[120];
static uint8_t send(ep_screen *screen, ep_record record, uint64_t now) {
    ep_record reply;
    size_t size = ep_screen_handle(screen, &record, now, response, sizeof response);
    assert(ep_wire_decode(response, size, &reply) && reply.kind == 9);
    assert(reply.epoch == record.epoch && reply.id == record.id && reply.payload[0] == record.kind);
    assert(ep_filled(reply.payload + 9, 3, 0));
    return reply.payload[1];
}
static ep_record transaction(uint8_t kind, uint64_t id) {
    return (ep_record){.kind=kind, .epoch=1, .id=id, .size=32, .payload=digest};
}
static ep_record chunk(uint64_t id, uint8_t pass, uint32_t offset, uint16_t size) {
    return (ep_record){.kind=5, .epoch=1, .id=id, .pass=pass, .offset=offset, .size=size, .payload=pixels};
}
static void fresh_profile(ep_screen *screen, ep_screen_config config) {
    begins = writes = commits = aborts = 0;
    fail_begin = fail_write = fail_commit = fail_abort = false;
    memset(pixels, 0, sizeof pixels);
    memcpy(claim, boot, 16); memset(claim + 16, 0x67, 16);
    assert(mbedtls_sha256(pixels, sizeof pixels, digest, 0) == 0);
    ep_sink sink = {NULL, begin, write_pixels, commit, abort_pixels};
    assert(ep_screen_init(screen, config, sink, boot, identity));
    assert(send(screen, (ep_record){.kind=2, .payload=claim, .size=32}, 0) == 0);
    assert(send(screen, (ep_record){.kind=3, .epoch=1, .payload=claim, .size=32}, 0) == 0);
}
static void fresh(ep_screen *screen) {
    fresh_profile(screen, (ep_screen_config){800, 480, 1000, 1, 2, 1, 180000});
}
static void upload(ep_screen *screen, uint64_t now) {
    for (uint8_t pass = 0; pass < 2; pass++)
        for (uint32_t offset = 0; offset < sizeof pixels; offset += 1000)
            assert(send(screen, chunk(1, pass, offset, 1000), now) == 0);
    assert(screen->state == 2 && commits == 0);
}
static void chunk_faults(void) {
    for (unsigned fault = 0; fault < 5; fault++) {
        ep_screen screen; fresh(&screen);
        assert(send(&screen, transaction(4, 1), 180000) == 0);
        ep_record record = chunk(1, 0, 0, 1000);
        if (fault == 0) record.pass = 1;
        if (fault == 1) record.offset = 1;
        if (fault == 2) record.size = 0;
        if (fault == 3) record.size = 1001;
        if (fault == 4) {
            assert(send(&screen, record, 180001) == 0);
        }
        unsigned previous = writes;
        assert(send(&screen, record, 180001) == 8);
        assert(writes == previous && aborts == 1 && commits == 0);
        assert(send(&screen, transaction(4, 1), 180002) == 4);
    }
    ep_screen screen; fresh(&screen);
    assert(send(&screen, transaction(4, 1), 180000) == 0);
    for (uint32_t offset = 0; offset < 47000; offset += 1000)
        assert(send(&screen, chunk(1, 0, offset, 1000), 180001) == 0);
    assert(send(&screen, chunk(1, 0, 47000, 999), 180001) == 0);
    assert(send(&screen, chunk(1, 0, 47999, 2), 180001) == 8);
    assert(writes == 48 && aborts == 1 && commits == 0);
}
static void digest_and_sink_failures(void) {
    ep_screen screen; fresh(&screen); digest[0] ^= 1;
    assert(send(&screen, transaction(4, 1), 180000) == 0);
    for (uint32_t offset = 0; offset < 47000; offset += 1000)
        assert(send(&screen, chunk(1, 0, offset, 1000), 180001) == 0);
    assert(send(&screen, chunk(1, 0, 47000, 1000), 180001) == 9);
    assert(aborts == 1 && commits == 0 && screen.state == 4);
    for (unsigned fault = 0; fault < 3; fault++) {
        fresh(&screen);
        fail_begin = fault == 0;
        uint8_t code = send(&screen, transaction(4, 1), 180000);
        if (fault == 0) assert(code == 11);
        else {
            assert(code == 0);
            fail_write = fault == 1;
            fail_abort = fault == 2;
            ep_record record = chunk(1, 0, fault == 2 ? 1 : 0, 1000);
            assert(send(&screen, record, 180001) == 11);
        }
        assert(aborts == 1 && screen.fatal && commits == 0);
        assert(send(&screen, transaction(4, 2), 360000) == 11);
    }
    fresh(&screen); assert(send(&screen, transaction(4, 1), 180000) == 0); upload(&screen, 180001);
    fail_commit = true;
    assert(send(&screen, transaction(6, 1), 180002) == 11);
    assert(commits == 1 && aborts == 1 && !screen.current_image);
    assert(send(&screen, transaction(6, 1), 180003) != 0 && commits == 1);
}
static void replay_and_disconnect(void) {
    ep_screen screen; fresh(&screen);
    assert(send(&screen, transaction(4, 1), 180000) == 0); upload(&screen, 180001);
    assert(send(&screen, transaction(6, 1), 180002) == 0);
    assert(send(&screen, transaction(6, 1), 180003) == 0 && commits == 1);
    ep_screen_disconnect(&screen); assert(aborts == 0);
    assert(send(&screen, transaction(6, 1), 180004) == 2 && commits == 1);
    assert(send(&screen, (ep_record){.kind=3, .epoch=1, .payload=claim, .size=32}, 180005) == 0);
    assert(send(&screen, transaction(6, 1), 180006) == 0 && commits == 1);
    assert(send(&screen, transaction(4, 1), 360002) == 4);
    assert(send(&screen, transaction(4, 2), 360001) != 0); // Regressing clock cannot consume ID.
    assert(screen.consumed == 1);
    assert(send(&screen, transaction(4, 2), 360002) == 0);
    ep_screen_disconnect(&screen); assert(aborts == 1 && commits == 1);
    assert(send(&screen, (ep_record){.kind=3, .epoch=1, .payload=claim, .size=32}, 360003) == 0);
    assert(send(&screen, transaction(4, 2), 360004) == 4);
    assert(send(&screen, transaction(7, 2), 360005) == 0 && screen.state == 5);
}
static void query_and_timeout(void) {
    ep_screen screen; fresh(&screen);
    assert(send(&screen, transaction(7, 1), 180000) == 0);
    assert(response[EP_HEADER + 2] == 0);
    assert(send(&screen, transaction(4, 1), 180000) == 0);
    assert(send(&screen, chunk(1, 0, 0, 1000), 180001) == 0);
    assert(send(&screen, transaction(7, 2), 180002) == 0);
    assert(ep_filled(response + EP_HEADER + 2, 7, 0));
    uint8_t wrong[32]; memcpy(wrong, digest, 32); wrong[0] ^= 1;
    ep_record query = transaction(7, 1); query.payload = wrong;
    assert(send(&screen, query, 180003) == 5);
    assert(ep_filled(response + EP_HEADER + 2, 7, 0));
    assert(send(&screen, transaction(4, 2), 200001) == 10);
    assert(begins == 1 && screen.consumed == 1 && aborts == 1);
    assert(send(&screen, transaction(4, 2), 200002) == 0);
    assert(send(&screen, (ep_record){.kind=8, .epoch=1}, 200003) == 0);
    assert(send(&screen, transaction(7, 2), 200004) == 0 && screen.state == 5);
}
static void profile_padding_and_cadence(void) {
    ep_screen screen;
    const ep_screen_config config = {9, 2, 3, 7, 1, 42, 17};
    for (unsigned bad = 0; bad < 2; bad++) {
        fresh_profile(&screen, config);
        assert(send(&screen, (ep_record){.kind=1}, 0) == 0);
        const uint8_t *caps = response + EP_HEADER + 48;
        assert(ep_le(caps, 2) == 9 && ep_le(caps + 2, 2) == 2 && ep_le(caps + 4, 2) == 2);
        assert(ep_le(caps + 6, 2) == 3 && caps[8] == 1 && ep_le(caps + 12, 4) == 42);
        assert(ep_le(caps + 16, 2) == 7 && ep_le(caps + 20, 4) == 17);
        const uint8_t frame[] = {0x55, 0x80, 0xaa, 0};
        assert(mbedtls_sha256(frame, sizeof frame, digest, 0) == 0);
        assert(send(&screen, transaction(4, 1), 16) == 6 && begins == 0);
        assert(send(&screen, transaction(4, 1), 17) == 0);
        ep_record record = chunk(1, 0, 0, 1); record.payload = frame;
        assert(send(&screen, record, 18) == 0);
        uint8_t middle[] = {bad ? 0x81 : 0x80, 0xaa};
        record.offset = 1; record.size = 2; record.payload = middle;
        assert(send(&screen, record, 19) == (bad ? 8 : 0));
        if (bad) { assert(aborts == 1 && writes == 1); continue; }
        record.offset = 3; record.size = 1; record.payload = frame + 3;
        assert(send(&screen, record, 20) == 0 && screen.state == 2 && screen.pass == 1);
        assert(send(&screen, transaction(6, 1), 21) == 0);
        assert(send(&screen, transaction(4, 2), 37) == 6);
        assert(ep_le(response + EP_HEADER + 20, 4) == 1);
        assert(send(&screen, transaction(4, 2), 38) == 0 && begins == 2);
        ep_screen_disconnect(&screen);
    }
}
static void total_timeout_and_packed_snapshot(void) {
    ep_screen screen; fresh(&screen);
    assert(send(&screen, transaction(4, 1), 180000) == 0);
    // Cached hardware queries must not cause timeout-driven sink operations.
    assert(send(&screen, (ep_record){.kind=10}, 999999) == 12);
    assert(send(&screen, (ep_record){.kind=12}, 999999) == 12);
    assert(screen.observed == 180000 && aborts == 0);
    for (uint32_t i = 0; i < 6; i++)
        assert(send(&screen, chunk(1, 0, i, 1), 180000 + (uint64_t)(i + 1) * 19000) == 0);
    assert(ep_screen_tick(&screen, 300000) == 10 && aborts == 1);
    fresh(&screen); assert(send(&screen, transaction(4, 1), 180000) == 0);
    ep_record packed = chunk(1, 0, 0, 1); packed.kind = 11;
    assert(send(&screen, packed, 180001) == 8);
    assert(response[EP_HEADER + 2] == 4 && aborts == 1 && writes == 0);
    fresh(&screen); assert(send(&screen, transaction(4, 1), 180000) == 0); upload(&screen, 180001);
    assert(send(&screen, chunk(1, 1, 0, 1), 180002) == 7 && aborts == 1 && commits == 0);
}
int main(void) {
    chunk_faults(); digest_and_sink_failures(); replay_and_disconnect(); query_and_timeout();
    profile_padding_and_cadence(); total_timeout_and_packed_snapshot();
    puts("screen faults: chunk integrity, sink failure, replay, query and timeout contracts passed");
}
