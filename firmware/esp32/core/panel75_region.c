#include "panel75.h"
#include "panel75_window.h"

static bool encode(const ep_region *r, uint8_t window[9]) {
    uint32_t bytes;
    return r && ep_panel75_encode_window(
        (ep_panel75_window){r->left, r->top, r->right, r->bottom}, window, &bytes) && bytes == r->bytes;
}
static bool valid(void *context, const ep_region *r) {
    (void)context;
    uint8_t window[9];
    return encode(r, window);
}
static int begin(void *context, const ep_region *r) {
    ep_panel75 *p = context;
    uint8_t window[9];
    if (!encode(r, window) || p->active) return -1;
    if (ep_panel75_start(p)) return -1;
    p->partial = true; p->plane_bytes = r->bytes;
    // Pinned Waveshare Init_Part/Display_Part; reset exits deep sleep.
    // Both OLD (10h) and NEW (13h) region planes are supplied by the receiver.
    // A9h selects DDX=01: invert canonical 1=black on BOTH passes (R50h).
    // Sources and the unqualified sleep/wake boundary: docs/esp32-partial-research.md.
    const uint8_t mode = 0x1f, cascade = 2, temperature = 0x6e, interval[] = {0xa9, 7};
    if (ep_panel_command(p, 0, &mode, 1) || ep_panel_command(p, 4, NULL, 0)) return -1;
    p->io.delay_us(p->io.context, 100000);
    if (ep_panel_ready(p, 10000000)) return -1;
    return ep_panel_command(p, 0xe0, &cascade, 1) || ep_panel_command(p, 0xe5, &temperature, 1) ||
        ep_panel_command(p, 0x50, interval, sizeof interval) || ep_panel_command(p, 0x91, NULL, 0) ||
        ep_panel_command(p, 0x90, window, sizeof window) ? -1 : 0;
}
ep_region_sink ep_panel75_regions(void) { return (ep_region_sink){valid, begin}; }
