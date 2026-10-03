#include "panel75.h"
#include <assert.h>
#include <string.h>

typedef struct {
    uint64_t now;
    bool dc, cs;
    unsigned count, calls, fail_at, refreshes, sleeps;
    uint8_t command, commands[32], data[32][9], sizes[32], pixels[2];
    uint32_t planes[2];
} fake_io;
static fake_io fake;
static bool fault(void) { return ++fake.calls == fake.fail_at; }
static int pin(void *ctx, ep_panel_pin name, bool high) {
    (void)ctx;
    if (fault()) return -1;
    if (name == EP_DC) fake.dc = high;
    if (name == EP_CS) fake.cs = high;
    return 0;
}
static int spi(void *ctx, const uint8_t *p, size_t n) {
    (void)ctx; assert(!fake.cs && p && n);
    if (fault()) return -1;
    if (!fake.dc) {
        assert(n == 1 && fake.count < 32);
        fake.command = p[0]; fake.commands[fake.count++] = p[0];
        if (p[0] == 0x12) fake.refreshes++;
        if (p[0] == 7) fake.sleeps++;
    } else if (fake.command == 0x10 || fake.command == 0x13) {
        unsigned pass = fake.command == 0x10 ? 0 : 1;
        assert(n <= 64);
        for (size_t i = 0; i < n; i++) assert(p[i] == fake.pixels[pass]);
        fake.planes[pass] += (uint32_t)n;
    } else {
        assert(fake.count && n <= 9);
        memcpy(fake.data[fake.count - 1], p, n); fake.sizes[fake.count - 1] = (uint8_t)n;
    }
    return 0;
}
static int busy(void *ctx) { (void)ctx; return fault() ? -1 : 1; }
static uint64_t now(void *ctx) { (void)ctx; return fake.now; }
static void delay(void *ctx, uint32_t us) { (void)ctx; fake.now += us; }
static ep_sink fresh(ep_panel75 *p) {
    memset(&fake, 0, sizeof fake); fake.cs = true;
    fake.pixels[0] = 0xc3; fake.pixels[1] = 0x5a;
    return ep_panel75_create(p, (ep_panel_io){NULL, spi, pin, busy, now, delay});
}
static ep_region area(void) {
    return (ep_region){.left=240, .top=240, .right=272, .bottom=280, .bytes=160};
}
static void stage(ep_sink sink) {
    uint8_t data[160];
    for (uint8_t pass = 0; pass < 2; pass++) {
        memset(data, pass ? 0xa5 : 0x3c, sizeof data);
        assert(sink.write(sink.context, pass, 0, data, 100) == 0);
        assert(sink.write(sink.context, pass, 100, data + 100, 60) == 0);
    }
}
static void source_sequence(void) {
    ep_panel75 p; ep_sink sink = fresh(&p);
    ep_region_sink region = ep_panel75_regions(); ep_region r = area();
    assert(region.valid(sink.context, &r));
    assert(region.begin(sink.context, &r) == 0);
    assert(p.phase == 1 && p.step == 21 && p.command == 0x90);
    const uint8_t commands[] = {0, 4, 0xe0, 0xe5, 0x50, 0x91, 0x90};
    const uint8_t sizes[] = {1, 0, 1, 1, 2, 0, 9};
    const uint8_t data[][9] = {{0x1f}, {0}, {2}, {0x6e}, {0xa9,7}, {0},
        {0,240,1,15,0,240,1,23,1}};
    assert(fake.count == sizeof commands && !memcmp(fake.commands, commands, sizeof commands));
    for (size_t i = 0; i < sizeof commands; i++) {
        assert(fake.sizes[i] == sizes[i] && !memcmp(fake.data[i], data[i], sizes[i]));
    }
    assert(sink.commit(sink.context) != 0 && !fake.refreshes);
    stage(sink);
    assert(fake.planes[0] == 160 && fake.planes[1] == 160 && !fake.refreshes);
    assert(sink.commit(sink.context) == 0 && fake.refreshes == 1 && fake.sleeps == 1);
    assert(!p.active && fake.cs);
    assert(sink.commit(sink.context) != 0 && fake.refreshes == 1);
}
static void rejected_geometry(void) {
    ep_panel75 p; ep_sink sink = fresh(&p);
    ep_region_sink adapter = ep_panel75_regions(); ep_region r = area();
    assert(!adapter.valid(sink.context, NULL));
    r.bytes++;
    assert(!adapter.valid(sink.context, &r) && adapter.begin(sink.context, &r) != 0);
    assert(!fake.calls && !p.active);
    r = area(); assert(adapter.begin(sink.context, &r) == 0);
    unsigned calls = fake.calls;
    assert(adapter.begin(sink.context, &r) != 0 && fake.calls == calls);
    assert(sink.abort(sink.context) == 0 && !fake.refreshes && fake.sleeps == 1);
    assert(sink.begin(sink.context) == 0 && !p.partial && p.plane_bytes == 48000);
    assert(sink.abort(sink.context) == 0 && !fake.refreshes);
}
static void begin_faults(void) {
    ep_panel75 p; ep_sink sink = fresh(&p);
    ep_region_sink adapter = ep_panel75_regions(); ep_region r = area();
    assert(adapter.begin(sink.context, &r) == 0); unsigned steps = fake.calls;
    for (unsigned fail = 1; fail <= steps; fail++) {
        sink = fresh(&p); fake.fail_at = fail;
        assert(adapter.begin(sink.context, &r) != 0 && !fake.refreshes);
        fake.fail_at = 0;
        assert(sink.abort(sink.context) == 0 && !p.active && fake.cs && !fake.refreshes);
    }
}
static void commit_faults(void) {
    ep_panel75 p; ep_sink sink = fresh(&p);
    ep_region_sink adapter = ep_panel75_regions(); ep_region r = area();
    assert(adapter.begin(sink.context, &r) == 0); stage(sink);
    unsigned before = fake.calls;
    assert(sink.commit(sink.context) == 0); unsigned steps = fake.calls - before;
    for (unsigned fail = 1; fail <= steps; fail++) {
        sink = fresh(&p); assert(adapter.begin(sink.context, &r) == 0); stage(sink);
        fake.fail_at = fake.calls + fail;
        assert(sink.commit(sink.context) != 0 && p.failed);
        unsigned refreshes = fake.refreshes;
        fake.fail_at = 0;
        assert(sink.commit(sink.context) != 0 && fake.refreshes == refreshes);
        assert(sink.abort(sink.context) == 0 && !p.active && fake.cs);
        assert(fake.refreshes == refreshes);
    }
}
static void request(ep_screen *s, uint8_t kind, uint64_t id, uint8_t pass, uint32_t offset,
                    const uint8_t *data, uint16_t n) {
    ep_record r = {.kind=kind, .epoch=1, .id=id, .pass=pass, .offset=offset, .payload=data, .size=n}, reply;
    uint8_t out[120];
    size_t size = ep_screen_handle(s, &r, fake.now / 1000 + 1, out, sizeof out);
    assert(ep_wire_decode(out, size, &reply) && reply.payload[1] == 0);
}
static void receiver_to_spi(void) {
    ep_panel75 panel; ep_sink sink = fresh(&panel);
    ep_screen s; ep_screen_config config = {800, 480, 1000, 1, 2, 1, 1};
    uint8_t boot[16] = {1}, device[16] = {2}, full[48000], digest[32], p[148] = {0};
    memset(full, 0x3c, sizeof full);
    assert(ep_screen_init(&s, config, sink, boot, device));
    assert(ep_screen_regions(&s, ep_panel75_regions()));
    s.generation = 1; s.bound = true;
    assert(mbedtls_sha256(full, sizeof full, digest, 0) == 0);
    fake.pixels[1] = 0x3c;
    request(&s, 4, 1, 0, 0, digest, 32);
    for (uint8_t pass = 0; pass < 2; pass++)
        for (uint32_t offset = 0; offset < sizeof full; offset += 1000)
            request(&s, 5, 1, pass, offset, full + offset, 1000);
    request(&s, 6, 1, 0, 0, digest, 32);
    assert(s.current_image && fake.refreshes == 1 && fake.sleeps == 1);
    memcpy(p + 32, digest, 32);
    assert(mbedtls_sha256(full, 160, p + 64, 0) == 0);
    memset(full, 0xa5, 160);
    assert(mbedtls_sha256(full, 160, p + 96, 0) == 0);
    ep_put_le(p + 128, 240, 2); ep_put_le(p + 130, 240, 2);
    ep_put_le(p + 132, 272, 2); ep_put_le(p + 134, 280, 2);
    ep_put_le(p + 140, 1, 4); ep_put_le(p + 144, 1, 4);
    uint8_t signed_data[131] = "EPS2-region-v1";
    memcpy(signed_data + 15, p + 32, 116);
    assert(mbedtls_sha256(signed_data, sizeof signed_data, p, 0) == 0);
    fake.pixels[1] = 0x5a;
    request(&s, 14, 2, 0, 0, p, sizeof p);
    request(&s, 5, 2, 0, 0, full + 160, 160);
    request(&s, 5, 2, 1, 0, full, 160);
    assert(fake.refreshes == 1 && s.state == 2);
    request(&s, 6, 2, 0, 0, p, 32);
    request(&s, 6, 2, 0, 0, p, 32);
    assert(s.current_image && fake.refreshes == 2 && fake.sleeps == 2);
    assert(fake.planes[0] == 48160 && fake.planes[1] == 48160);
    mbedtls_sha256_free(&s.hash);
}
int main(void) {
    source_sequence(); rejected_geometry(); begin_faults(); commit_faults(); receiver_to_spi();
}
