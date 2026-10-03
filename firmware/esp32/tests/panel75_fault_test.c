#include "panel75.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

typedef struct {
    uint64_t now, reset_times[3];
    bool cs, dc, reset_values[3], stuck, persistent_spi, regress;
    unsigned calls, fail_at, spi_calls, resets, refreshes, sleeps, releases, clocks;
    uint32_t planes[2];
    uint8_t command, commands[32], params[32][8], lengths[32];
    unsigned command_count;
} fake_io;
static fake_io fake;
static bool fault(fake_io *f) { return ++f->calls == f->fail_at; }
static int pin(void *context, ep_panel_pin name, bool high) {
    fake_io *f = context;
    if (name == EP_CS && high) f->releases++;
    if (fault(f)) return -1;
    if (name == EP_CS) f->cs = high;
    if (name == EP_DC) f->dc = high;
    if (name == EP_RESET && f->resets < 3) {
        f->reset_values[f->resets] = high;
        f->reset_times[f->resets++] = f->now;
    }
    return 0;
}
static int spi(void *context, const uint8_t *data, size_t size) {
    fake_io *f = context;
    assert(!f->cs && data && size);
    f->spi_calls++;
    if (fault(f) || f->persistent_spi) return -1;
    if (!f->dc) {
        assert(size == 1 && f->command_count < 32);
        f->command = data[0]; f->commands[f->command_count++] = data[0];
        if (f->command == 0x12) f->refreshes++;
        if (f->command == 0x07) f->sleeps++;
    } else if (f->command == 0x10 || f->command == 0x13) {
        unsigned plane = f->command == 0x10 ? 0 : 1;
        assert(size <= 64); // Matches the original ESP32 platform's non-DMA SPI limit.
        for (size_t i = 0; i < size; i++) assert(data[i] == (plane ? 0xa5 : 0x5a));
        f->planes[plane] += (uint32_t)size;
    } else {
        assert(f->command_count && size <= 8);
        unsigned position = f->command_count - 1;
        memcpy(f->params[position], data, size); f->lengths[position] = (uint8_t)size;
    }
    return 0;
}
static int busy(void *context) {
    fake_io *f = context;
    if (fault(f)) return -1;
    return f->stuck ? 0 : 1;
}
static uint64_t now_us(void *context) {
    fake_io *f = context;
    f->clocks++;
    return f->regress ? (f->clocks == 1 ? 100 : 99) : f->now;
}
static void delay_us(void *context, uint32_t time) { ((fake_io *)context)->now += time; }
static ep_sink fresh(ep_panel75 *panel) {
    memset(&fake, 0, sizeof fake); fake.cs = true;
    ep_panel_io io = {&fake, spi, pin, busy, now_us, delay_us};
    return ep_panel75_create(panel, io);
}
static void pass(ep_sink sink, uint8_t plane) {
    uint8_t pixels[1000]; memset(pixels, 0xa5, sizeof pixels);
    for (uint32_t offset = 0; offset < 48000; offset += 1000)
        assert(sink.write(sink.context, plane, offset, pixels, sizeof pixels) == 0);
}
static void source_sequence(void) {
    ep_panel75 panel; ep_sink sink = fresh(&panel);
    assert(sink.begin(sink.context) == 0);
    assert(fake.resets == 3 && fake.reset_values[0] && !fake.reset_values[1] && fake.reset_values[2]);
    assert(fake.reset_times[1] - fake.reset_times[0] == 20000);
    assert(fake.reset_times[2] - fake.reset_times[1] == 2000);
    assert(fake.now - fake.reset_times[2] >= 120000);
    const uint8_t commands[] = {1, 6, 4, 0, 0x61, 0x15, 0x50, 0x60};
    const uint8_t lengths[] = {4, 4, 0, 1, 4, 1, 2, 1};
    const uint8_t params[][8] = {{7,7,0x3f,0x3f}, {0x17,0x17,0x28,0x17}, {0}, {0x1f},
        {3,0x20,1,0xe0}, {0}, {0x10,7}, {0x22}};
    assert(fake.command_count == sizeof commands && !memcmp(fake.commands, commands, sizeof commands));
    for (unsigned i = 0; i < sizeof commands; i++) {
        assert(fake.lengths[i] == lengths[i]);
        assert(!memcmp(fake.params[i], params[i], lengths[i]));
    }
    assert(sink.commit(sink.context) != 0 && fake.refreshes == 0);
    pass(sink, 0); assert(sink.commit(sink.context) != 0 && fake.refreshes == 0);
    pass(sink, 1); assert(fake.planes[0] == 48000 && fake.planes[1] == 48000 && fake.refreshes == 0);
    assert(sink.commit(sink.context) == 0 && fake.refreshes == 1 && fake.sleeps == 1 && !panel.active);
    assert(sink.commit(sink.context) != 0 && fake.refreshes == 1);
    assert(sink.abort(sink.context) == 0 && fake.sleeps == 1);
}
static void begin_faults(void) {
    ep_panel75 panel; ep_sink sink = fresh(&panel);
    assert(sink.begin(sink.context) == 0); unsigned steps = fake.calls;
    for (unsigned failed = 1; failed <= steps; failed++) {
        sink = fresh(&panel); fake.fail_at = failed;
        assert(sink.begin(sink.context) != 0 && fake.refreshes == 0);
        fake.fail_at = 0;
        assert(sink.abort(sink.context) == 0 && !panel.active && fake.cs);
        assert(fake.refreshes == 0);
    }
}
static void transfer_cleanup(void) {
    for (unsigned failed = 1; failed <= 4; failed++) {
        ep_panel75 panel; fresh(&panel); fake.fail_at = failed; fake.command_count = 1;
        uint8_t byte = 0x11;
        assert(ep_panel_data(&panel, &byte, 1) != 0);
        assert(fake.releases == 1);
        if (failed != 4) assert(fake.cs);
        assert(fake.spi_calls == (failed >= 3 ? 1u : 0u));
    }
}
static void staging_faults(void) {
    uint8_t pixels[1000]; memset(pixels, 0xa5, sizeof pixels);
    for (unsigned failure = 0; failure < 5; failure++) {
        ep_panel75 panel; ep_sink sink = fresh(&panel);
        assert(sink.begin(sink.context) == 0);
        unsigned before = fake.spi_calls;
        uint8_t plane = failure == 0 ? 1 : 0;
        uint32_t offset = failure == 1 ? UINT32_MAX : 0;
        size_t size = failure == 2 ? 0 : (failure == 3 ? 48001 : sizeof pixels);
        if (failure == 4) fake.fail_at = fake.calls + 7; // First pixel SPI after plane command.
        assert(sink.write(sink.context, plane, offset, pixels, size) != 0);
        if (failure < 4) assert(fake.spi_calls == before);
        assert(panel.offset == 0 && panel.pass == 0 && fake.refreshes == 0);
        fake.fail_at = 0;
        assert(sink.abort(sink.context) == 0 && !panel.active && fake.cs);
        assert(sink.commit(sink.context) != 0 && fake.refreshes == 0);
        assert(sink.write(sink.context, 0, 0, pixels, 1) != 0);
    }
}
static void rail_failure_and_deadlines(void) {
    ep_panel75 panel; ep_sink sink = fresh(&panel);
    assert(sink.begin(sink.context) == 0);
    fake.stuck = true;
    uint64_t before = fake.now;
    assert(sink.abort(sink.context) != 0 && panel.active);
    uint8_t diagnostic[11]; ep_panel_diagnostic(&panel, diagnostic);
    assert(diagnostic[0] == 1 && diagnostic[1] == 2 && diagnostic[2] == 6);
    assert(diagnostic[4] == 2 && diagnostic[5] == 1);
    assert(fake.now - before <= 10000000 && fake.refreshes == 0 && fake.sleeps == 0 && fake.cs);
    fake.stuck = false; fake.persistent_spi = true;
    assert(sink.abort(sink.context) != 0 && panel.active && fake.cs);
    assert(fake.refreshes == 0 && fake.sleeps == 0); // Physical rail cutoff is not claimed.
    fake.persistent_spi = false;
    assert(sink.abort(sink.context) == 0 && !panel.active);
    uint8_t after_cleanup[11]; ep_panel_diagnostic(&panel, after_cleanup);
    assert(!memcmp(diagnostic, after_cleanup, sizeof diagnostic));
    fresh(&panel); fake.stuck = true;
    assert(ep_panel_ready(&panel, 1) != 0 && fake.now == 0);
    fresh(&panel); fake.regress = true;
    assert(ep_panel_ready(&panel, 10000) != 0);
    fresh(&panel); fake.fail_at = 1;
    assert(ep_panel_ready(&panel, 10000) != 0 && fake.now == 5000);
    fresh(&panel);
    assert(ep_panel_ready(&panel, 10000) == 0 && fake.now == 10000);
}
static void commit_cleanup_faults(void) {
    ep_panel75 panel; ep_sink sink = fresh(&panel);
    assert(sink.begin(sink.context) == 0); pass(sink, 0); pass(sink, 1);
    unsigned before = fake.calls;
    assert(sink.commit(sink.context) == 0);
    unsigned steps = fake.calls - before;
    for (unsigned failed = 1; failed <= steps; failed++) {
        sink = fresh(&panel);
        assert(sink.begin(sink.context) == 0); pass(sink, 0); pass(sink, 1);
        fake.fail_at = fake.calls + failed;
        assert(sink.commit(sink.context) != 0 && panel.active);
        unsigned refreshed = fake.refreshes;
        assert(refreshed <= 1);
        // A late failure may follow physical refresh. Owner aborts, never retries Commit.
        fake.fail_at = 0;
        assert(sink.abort(sink.context) == 0 && !panel.active && fake.cs);
        assert(fake.refreshes == refreshed);
        assert(sink.commit(sink.context) != 0 && fake.refreshes == refreshed);
    }
}
int main(void) {
    source_sequence(); begin_faults(); transfer_cleanup(); staging_faults(); rail_failure_and_deadlines();
    commit_cleanup_faults();
    puts("panel75 faults: pinned sequence, reset, CS cleanup, staging and rail-failure reporting passed");
}
