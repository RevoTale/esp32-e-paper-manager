#include "screen.h"
#include <assert.h>
#include <string.h>

static unsigned fulls, regions, writes, commits, aborts;
static unsigned sink_failure;
static int full(void *ctx) { (void)ctx; fulls++; return 0; }
static bool valid_region(void *ctx, const ep_region *r) {
    (void)ctx; return r->right - r->left >= 16 && r->bottom - r->top >= 2;
}
static int region(void *ctx, const ep_region *r) {
    (void)ctx; assert(r->bytes == 4); regions++; return sink_failure == 1 ? -1 : 0;
}
static int write_pixels(void *ctx, uint8_t pass, uint32_t offset, const uint8_t *p, size_t n) {
    (void)ctx; assert(pass < 2 && offset == 0 && p && (n == 4 || n == 8));
    writes++; return sink_failure == 2 ? -1 : 0;
}
static int commit(void *ctx) { (void)ctx; commits++; return sink_failure == 3 ? -1 : 0; }
static int abort_pixels(void *ctx) { (void)ctx; aborts++; return sink_failure == 4 ? -1 : 0; }
static uint8_t send(ep_screen *s, uint8_t kind, uint64_t id, uint8_t pass,
                    const uint8_t *p, uint16_t n) {
    ep_record r = {.kind=kind, .epoch=1, .id=id, .pass=pass, .size=n, .payload=p}, reply;
    uint8_t out[120];
    size_t size = ep_screen_handle(s, &r, s->observed + 1, out, sizeof out);
    assert(ep_wire_decode(out, size, &reply)); return reply.payload[1];
}
static void setup(ep_screen *s, const uint8_t *pixels, uint8_t *digest) {
    fulls = regions = writes = commits = aborts = sink_failure = 0;
    ep_sink sink = {NULL, full, write_pixels, commit, abort_pixels};
    ep_screen_config config = {16, 2, 4, 1, 2, 1, 1};
    uint8_t boot[16] = {1}, device[16] = {2};
    assert(ep_screen_init(s, config, sink, boot, device));
    assert(ep_screen_regions(s, (ep_region_sink){valid_region, region}));
    // Lease mechanics are covered separately; exercise the validated frame path.
    s->generation = 1; s->bound = true;
    assert(mbedtls_sha256(pixels, 4, digest, 0) == 0);
    assert(send(s, 4, 1, 0, digest, 32) == 0);
    assert(send(s, 5, 1, 0, pixels, 4) == 0);
    assert(send(s, 5, 1, 1, pixels, 4) == 0);
    assert(send(s, 6, 1, 0, digest, 32) == 0);
}
static void seal_payload(uint8_t *p) {
    uint8_t input[131] = "EPS2-region-v1";
    memcpy(input + 15, p + 32, 116);
    assert(mbedtls_sha256(input, sizeof input, p, 0) == 0);
}
static void payload(uint8_t *p, const uint8_t *base, const uint8_t *old, const uint8_t *next) {
    memset(p, 0, 148); memcpy(p + 32, base, 32);
    assert(mbedtls_sha256(old, 4, p + 64, 0) == 0);
    assert(mbedtls_sha256(next, 4, p + 96, 0) == 0);
    ep_put_le(p + 132, 16, 2); ep_put_le(p + 134, 2, 2);
    ep_put_le(p + 140, 1, 4); ep_put_le(p + 144, 1, 4);
    seal_payload(p);
}
static void round_trip(void) {
    ep_screen s;
    uint8_t old[4] = {0, 1, 2, 3}, next[4] = {4, 5, 6, 7}, digest[32], p[148];
    setup(&s, old, digest); payload(p, digest, old, next);
    assert(send(&s, 14, 2, 0, p, sizeof p) == 0 && regions == 1);
    assert(send(&s, 5, 2, 0, old, 4) == 0);
    assert(send(&s, 5, 2, 1, next, 4) == 0 && s.state == 2);
    assert(send(&s, 6, 2, 0, p, 32) == 0 && commits == 2 && s.current_image);
    assert(send(&s, 6, 2, 0, p, 32) == 0 && commits == 2);
    assert(fulls == 1 && writes == 4 && !aborts);
    assert(send(&s, 14, 3, 0, p, sizeof p) == 5 && regions == 1); // stale baseline
    assert(!ep_screen_regions(&s, (ep_region_sink){valid_region, region}));
    mbedtls_sha256_free(&s.hash);
}
static void corrupt_planes(void) {
    for (uint8_t pass = 0; pass < 2; pass++) {
        ep_screen s;
        uint8_t old[4] = {0, 1, 2, 3}, next[4] = {4, 5, 6, 7}, digest[32], p[148], bad[4] = {255};
        setup(&s, old, digest); payload(p, digest, old, next);
        assert(send(&s, 14, 2, 0, p, sizeof p) == 0);
        if (pass) assert(send(&s, 5, 2, 0, old, 4) == 0);
        assert(send(&s, 5, 2, pass, bad, 4) == 9);
        assert(aborts == 1 && commits == 1 && s.state == 4 && !s.current_image);
        assert(send(&s, 6, 2, 0, p, 32) == 7 && commits == 1);
        assert(send(&s, 14, 3, 0, p, sizeof p) == 5 && regions == 1);
        // A failed region cannot silently retain a baseline; a full upload can restore it.
        assert(send(&s, 4, 3, 0, digest, 32) == 0 && !s.region_mode);
        assert(send(&s, 5, 3, 0, old, 4) == 0);
        assert(send(&s, 5, 3, 1, old, 4) == 0);
        assert(send(&s, 6, 3, 0, digest, 32) == 0 && commits == 2);
        assert(send(&s, 14, 4, 0, p, sizeof p) == 0);
        ep_screen_disconnect(&s);
        assert(aborts == 2 && !s.current_image && !s.bound);
        mbedtls_sha256_free(&s.hash);
    }
}
static void interrupted_and_timed_begin(void) {
    ep_screen s;
    uint8_t old[4] = {0}, next[4] = {1}, digest[32], p[148];
    setup(&s, old, digest); payload(p, digest, old, next);
    assert(send(&s, 14, 2, 0, p, sizeof p) == 0);
    uint64_t entered = s.observed;
    ep_screen_completed(&s, 14, 0, entered, entered + 1000);
    assert(s.progress == entered + 1000 && s.started == entered + 1000);
    assert(send(&s, 14, 3, 0, p, sizeof p) == 3 && regions == 1);
    assert(send(&s, 6, 2, 0, p, 32) == 7 && commits == 1);
    ep_screen_disconnect(&s);
    s.bound = true; // same lease rebound; incomplete base must remain unusable
    assert(send(&s, 14, 3, 0, p, sizeof p) == 5 && regions == 1);
    assert(aborts == 1 && commits == 1);
    mbedtls_sha256_free(&s.hash);
}
static void sink_faults(void) {
    for (unsigned fault = 1; fault <= 4; fault++) {
        ep_screen s;
        uint8_t old[4] = {0}, next[4] = {1}, digest[32], p[148];
        setup(&s, old, digest); payload(p, digest, old, next);
        sink_failure = fault;
        uint8_t code = send(&s, 14, 2, 0, p, sizeof p);
        if (fault != 1) {
            assert(code == 0);
            code = send(&s, 5, 2, 0, old, 4);
            if (fault != 2) {
                assert(code == 0);
                assert(send(&s, 5, 2, 1, next, 4) == 0);
                code = fault == 3 ? send(&s, 6, 2, 0, p, 32) : send(&s, 8, 0, 0, NULL, 0);
            }
        }
        assert(code == 11 && s.fatal && !s.current_image && aborts == 1);
        unsigned before = commits;
        assert(send(&s, 14, 3, 0, p, sizeof p) == 11 && regions == 1);
        assert(send(&s, 6, 2, 0, p, 32) == 7 && commits == before);
        mbedtls_sha256_free(&s.hash);
    }
}
static void admission_preserves_baseline(void) {
    ep_screen s;
    uint8_t old[4] = {0}, next[4] = {1}, digest[32], p[148];
    setup(&s, old, digest); payload(p, digest, old, next);
    p[0] ^= 1;
    assert(send(&s, 14, 2, 0, p, sizeof p) == 12);
    p[0] ^= 1;
    ep_put_le(p + 132, 8, 2); seal_payload(p); // generic byte alignment is not panel support
    assert(send(&s, 14, 2, 0, p, sizeof p) == 12);
    ep_put_le(p + 132, 16, 2);
    ep_put_le(p + 140, 1000, 4); seal_payload(p);
    assert(send(&s, 14, 2, 0, p, sizeof p) == 6);
    assert(!regions && !aborts && s.consumed == 1 && s.transaction == 1);
    assert(s.current_image && s.state == 3 && !memcmp(s.digest, digest, 32));
    p[136] = 1; seal_payload(p); // urgent uses its own interval, same pixel/base checks
    assert(send(&s, 14, 2, 0, p, sizeof p) == 0 && regions == 1);
    ep_screen_tick(&s, s.progress + 20000);
    assert(s.state == 4 && s.failure == 10 && aborts == 1 && commits == 1);
    assert(send(&s, 14, 3, 0, p, sizeof p) == 5);
    mbedtls_sha256_free(&s.hash);
}
static void region_extent_not_full_frame(void) {
    for (unsigned oversized = 0; oversized < 2; oversized++) {
        ep_screen s;
        ep_sink sink = {NULL, full, write_pixels, commit, abort_pixels};
        ep_screen_config config = {16, 4, 8, 1, 2, 1, 1};
        uint8_t boot[16] = {1}, device[16] = {2}, pixels[8] = {0}, next[4] = {1}, digest[32], p[148];
        fulls = regions = writes = commits = aborts = sink_failure = 0;
        assert(ep_screen_init(&s, config, sink, boot, device));
        assert(ep_screen_regions(&s, (ep_region_sink){valid_region, region}));
        s.generation = 1; s.bound = true;
        assert(mbedtls_sha256(pixels, sizeof pixels, digest, 0) == 0);
        assert(send(&s, 4, 1, 0, digest, 32) == 0);
        assert(send(&s, 5, 1, 0, pixels, 8) == 0);
        assert(send(&s, 5, 1, 1, pixels, 8) == 0);
        assert(send(&s, 6, 1, 0, digest, 32) == 0);
        payload(p, digest, pixels, next);
        assert(send(&s, 14, 2, 0, p, sizeof p) == 0);
        if (oversized) {
            assert(send(&s, 5, 2, 0, pixels, 8) == 8 && writes == 2 && aborts == 1);
        } else {
            assert(send(&s, 5, 2, 0, pixels, 4) == 0);
            assert(send(&s, 5, 2, 1, next, 4) == 0 && s.state == 2);
            assert(send(&s, 6, 2, 0, p, 32) == 0 && commits == 2);
        }
        mbedtls_sha256_free(&s.hash);
    }
}
int main(void) {
    round_trip();
    corrupt_planes();
    interrupted_and_timed_begin();
    sink_faults();
    admission_preserves_baseline();
    region_extent_not_full_frame();
}
