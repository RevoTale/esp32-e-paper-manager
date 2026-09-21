#include "panel75.h"
#include <assert.h>
#include <string.h>

static uint64_t now;
static bool dc, busy = true;
static uint8_t command;
static unsigned refreshes, sleeps, counts[2];
static int pin(void *ctx, ep_panel_pin p, bool high) { (void)ctx; if (p == EP_DC) dc = high; return 0; }
static int spi(void *ctx, const uint8_t *bytes, size_t n) {
    (void)ctx;
    if (!dc) {
        assert(n == 1); command = bytes[0];
        if (command == 0x12) refreshes++;
        if (command == 0x07) sleeps++;
    } else if (command == 0x10 || command == 0x13) {
        unsigned pass = command == 0x10 ? 0 : 1;
        for (size_t i = 0; i < n; i++) assert(bytes[i] == (pass ? 0xa5 : 0x5a));
        counts[pass] += (unsigned)n;
    }
    return 0;
}
static int read_busy(void *ctx) { (void)ctx; return busy ? 1 : 0; }
static uint64_t clock_us(void *ctx) { (void)ctx; return now; }
static void delay(void *ctx, uint32_t us) { (void)ctx; now += us; }
int main(void) {
    ep_panel75 panel;
    ep_panel_io io = {NULL, spi, pin, read_busy, clock_us, delay};
    ep_sink sink = ep_panel75_create(&panel, io);
    assert(sink.begin(sink.context) == 0);
    uint8_t pixels[1000]; memset(pixels, 0xa5, sizeof pixels);
    for (uint8_t pass = 0; pass < 2; pass++)
        for (uint32_t offset = 0; offset < 48000; offset += 1000)
            assert(sink.write(sink.context, pass, offset, pixels, sizeof pixels) == 0);
    assert(!refreshes && counts[0] == 48000 && counts[1] == 48000);
    assert(sink.commit(sink.context) == 0 && refreshes == 1 && sleeps == 1);
    assert(sink.abort(sink.context) == 0 && sleeps == 1);
    assert(sink.begin(sink.context) == 0);
    assert(sink.write(sink.context, 0, 0, pixels, 1000) == 0);
    assert(sink.abort(sink.context) == 0 && refreshes == 1 && sleeps == 2);
    busy = false;
    assert(sink.begin(sink.context) != 0 && panel.command == 0x04);
    uint64_t before = now;
    assert(sink.abort(sink.context) != 0 && now - before <= 10000000);
    assert(refreshes == 1 && sleeps == 2);
}
