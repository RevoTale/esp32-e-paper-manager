#include "panel75.h"
#include <string.h>

// Exact 7.5 V2 lifecycle, not the 1.54 tri-colour or 2.13 controller:
// https://github.com/waveshareteam/Pico_ePaper_Code/blob/c9bcd84db5adf5f085353649a8a5c31492bc5fb8/c/lib/e-Paper/EPD_7in5_V2.c
static int begin(void *context) {
    ep_panel75 *p = context;
    if (p->active) return -1;
    if (p->cycle < UINT32_MAX) p->cycle++;
    p->state = p->phase = 1; p->step = 3; p->failed = false;
    p->error_code = p->busy_flags = p->command = 0;
    p->started_us = p->io.now_us(p->io.context); p->elapsed_ms = 0;
    memset(p->samples, 0, sizeof p->samples); memset(p->low_samples, 0, sizeof p->low_samples);
    p->active = true; p->offset = 0; p->pass = 0;
    ep_panel_io *io = &p->io;
    if (io->pin(io->context, EP_CS, true) || io->pin(io->context, EP_DC, false)) return ep_panel_error(p);
    io->delay_us(io->context, 200000);
    if (io->pin(io->context, EP_RESET, true)) return ep_panel_error(p);
    io->delay_us(io->context, 20000);
    if (io->pin(io->context, EP_RESET, false)) return ep_panel_error(p);
    io->delay_us(io->context, 2000);
    if (io->pin(io->context, EP_RESET, true)) return ep_panel_error(p);
    io->delay_us(io->context, 20000);
    const uint8_t power[] = {7, 7, 0x3f, 0x3f}, boost[] = {0x17, 0x17, 0x28, 0x17};
    if (ep_panel_command(p, 1, power, sizeof power) ||
        ep_panel_command(p, 6, boost, sizeof boost) || ep_panel_command(p, 4, NULL, 0)) return -1;
    io->delay_us(io->context, 100000);
    if (ep_panel_ready(p, 10000000)) return -1;
    const uint8_t mode = 0x1f, size[] = {3, 0x20, 1, 0xe0}, zero = 0;
    const uint8_t interval[] = {0x10, 7}, tcon = 0x22;
    return ep_panel_command(p, 0, &mode, 1) || ep_panel_command(p, 0x61, size, sizeof size) ||
        ep_panel_command(p, 0x15, &zero, 1) || ep_panel_command(p, 0x50, interval, sizeof interval) ||
        ep_panel_command(p, 0x60, &tcon, 1) ? -1 : 0;
}
static int write_pixels(void *context, uint8_t pass, uint32_t offset, const uint8_t *bytes, size_t n) {
    ep_panel75 *p = context;
    if (!p->active || pass != p->pass || pass > 1 || offset != p->offset ||
        !n || n > 48000 - offset) return -1;
    if (!offset && ep_panel_command(p, pass ? 0x13 : 0x10, NULL, 0)) return -1;
    uint8_t scratch[64];
    for (size_t at = 0; at < n;) {
        size_t count = n - at < sizeof scratch ? n - at : sizeof scratch;
        for (size_t i = 0; i < count; i++) scratch[i] = pass ? bytes[at + i] : (uint8_t)~bytes[at + i];
        if (ep_panel_data(p, scratch, count)) return -1;
        at += count;
    }
    p->offset += (uint32_t)n;
    if (p->offset == 48000) { p->offset = 0; p->pass++; }
    return 0;
}
static int commit(void *context) {
    ep_panel75 *p = context;
    if (!p->active || p->pass != 2) return -1;
    if (ep_panel_command(p, 0x12, NULL, 0)) return -1;
    p->io.delay_us(p->io.context, 100000);
    if (ep_panel_ready(p, 30000000) || ep_panel_sleep(p)) return -1;
    p->state = 2;
    return 0;
}
static int abort_frame(void *context) {
    // Rev3 has no verified software rail switch. Sleep never sends Refresh;
    // on failure the owner must latch a fault and require physical power removal.
    return ep_panel_sleep(context);
}
ep_sink ep_panel75_create(ep_panel75 *p, ep_panel_io io) {
    memset(p, 0, sizeof *p); p->io = io;
    return (ep_sink){p, begin, write_pixels, commit, abort_frame};
}
