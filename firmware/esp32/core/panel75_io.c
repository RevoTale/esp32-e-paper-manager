#include "panel75.h"

static int transfer(ep_panel75 *p, bool data, const uint8_t *bytes, size_t n) {
    ep_panel_io *io = &p->io;
    int result = io->pin(io->context, EP_DC, data);
    if (!result) result = io->pin(io->context, EP_CS, false);
    if (!result) {
        result = io->spi(io->context, bytes, n);
        if (result && !p->failed) p->error_code = 3;
    }
    // Always release CS, including an SPI failure. Never hide cleanup failure.
    int release = io->pin(io->context, EP_CS, true);
    return result || release ? ep_panel_error(p) : 0;
}
int ep_panel_data(ep_panel75 *p, const uint8_t *bytes, size_t n) {
    return transfer(p, true, bytes, n);
}
int ep_panel_command(ep_panel75 *p, uint8_t command, const uint8_t *data, size_t n) {
    if (!p->failed) { p->command = command; p->busy_flags = 0; }
    ep_panel_trace_command(p, command);
    int result = transfer(p, false, &command, 1);
    return result || !n ? result : ep_panel_data(p, data, n);
}
int ep_panel_ready(ep_panel75 *p, uint32_t budget_us) {
    uint64_t start = p->io.now_us(p->io.context);
    if (budget_us < 10000) return ep_panel_error(p);
    p->io.delay_us(p->io.context, 5000);
    for (;;) {
        uint64_t now = p->io.now_us(p->io.context);
        if (now < start || now - start >= budget_us) {
            if (!p->failed) p->error_code = now < start ? 1 : 2;
            return ep_panel_error(p);
        }
        int busy = p->io.busy(p->io.context);
        if (busy < 0) return ep_panel_error(p);
        if (!p->failed) p->busy_flags = busy ? 3 : 1;
        unsigned wait = p->command == 4 ? 0 : (p->command == 0x12 ? 1 : 2);
        if (!p->failed && p->samples[wait] < UINT32_MAX) {
            p->samples[wait]++;
            if (!busy) p->low_samples[wait]++;
        }
        uint64_t remaining = budget_us - (now - start);
        if (busy == 1) {
            if (remaining < 5000) {
                if (!p->failed) p->error_code = 2;
                return ep_panel_error(p);
            }
            p->io.delay_us(p->io.context, 5000);
            return 0;
        }
        p->io.delay_us(p->io.context, remaining < 10000 ? (uint32_t)remaining : 10000);
    }
}
int ep_panel_sleep(ep_panel75 *p) {
    if (!p->active) return 0;
    if (!p->failed) p->phase = 6;
    const uint8_t border = 0xf7, sleep = 0xa5;
    if (ep_panel_command(p, 0x50, &border, 1) ||
        ep_panel_command(p, 0x02, NULL, 0) || ep_panel_ready(p, 10000000) ||
        ep_panel_command(p, 0x07, &sleep, 1)) return -1;
    p->active = false;
    p->state = 3;
    return 0;
}
