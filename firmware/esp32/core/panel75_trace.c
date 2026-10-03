#include "panel75.h"
#include <string.h>
static void elapsed(ep_panel75 *p) {
    uint64_t now = p->io.now_us(p->io.context);
    uint64_t ms = now >= p->started_us ? (now - p->started_us) / 1000 : 0;
    p->elapsed_ms = ms > UINT32_MAX ? UINT32_MAX : (uint32_t)ms;
}
int ep_panel_error(ep_panel75 *p) {
    if (!p->failed) {
        elapsed(p);
        if (!p->error_code) p->error_code = 1;
    }
    p->failed = true; p->state = 3;
    return -1;
}
void ep_panel_diagnostic(const ep_panel75 *p, uint8_t out[11]) {
    memset(out, 0, 11);
    if (!p->failed) return;
    out[0] = 1; out[1] = p->error_code; out[2] = p->phase;
    out[3] = p->step; out[4] = p->command; out[5] = p->busy_flags;
    ep_put_le(out + 7, UINT32_MAX, 4); // No invented byte-offset observation.
}
void ep_panel_trace_command(ep_panel75 *p, uint8_t command) {
    if (p->failed) return; // Preserve the first failure, not cleanup's last step.
    const uint8_t commands[] = {1, 6, 4, 0, 0x61, 0x15, 0x50, 0x60, 0x10, 0x13, 0x12, 2, 7,
        0xe0, 0xe5, 0x91, 0x90};
    const uint8_t phases[] = {1, 1, 2, 1, 1, 1, 1, 1, 3, 4, 5, 6, 6, 1, 1, 1, 1};
    const uint8_t steps[] = {4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 16, 17, 18, 19, 20, 21};
    bool power_off = command == 0x50 && p->phase >= 3;
    for (size_t i = 0; i < sizeof commands; i++) {
        if (command == commands[i]) { p->phase = phases[i]; p->step = steps[i]; break; }
    }
    if (power_off) { p->phase = 6; p->step = 15; }
    elapsed(p);
}
void ep_panel_status(const ep_panel75 *p, uint8_t out[36]) {
    memset(out, 0, 36); out[0] = 1;
    if (!p->cycle) return;
    out[1] = p->state; out[2] = p->phase; out[3] = p->step;
    ep_put_le(out + 4, p->cycle, 4); ep_put_le(out + 8, p->elapsed_ms, 4);
    for (unsigned i = 0; i < 3; i++) {
        ep_put_le(out + 12 + i * 8, p->samples[i], 4);
        ep_put_le(out + 16 + i * 8, p->low_samples[i], 4);
    }
}
