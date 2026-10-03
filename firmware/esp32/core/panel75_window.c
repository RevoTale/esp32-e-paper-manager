#include "panel75_window.h"
#include <string.h>

bool ep_panel75_encode_window(ep_panel75_window w, uint8_t out[9], uint32_t *bytes) {
    if (!out || !bytes || w.left >= w.right || w.top >= w.bottom ||
        w.right > 800 || w.bottom > 480 || w.left % 8 || w.right % 8) return false;
    // R90h requires the end bank/line to be strictly greater than the start.
    // https://files.waveshare.com/upload/6/60/7.5inch_e-Paper_V2_Specification.pdf#page=45
    if (w.right - w.left < 16 || w.bottom - w.top < 2) return false;

    // Waveshare's display_Partial uses inclusive controller endpoints:
    // https://github.com/waveshareteam/e-Paper/blob/master/RaspberryPi_JetsonNano/python/lib/waveshare_epd/epd7in5_V2.py
    // Subtract before splitting: an exclusive256 must encode00ff, not01ff.
    uint16_t right = (uint16_t)(w.right - 1u), bottom = (uint16_t)(w.bottom - 1u);
    const uint8_t encoded[9] = {
        (uint8_t)(w.left >> 8), (uint8_t)w.left,
        (uint8_t)(right >> 8), (uint8_t)right,
        (uint8_t)(w.top >> 8), (uint8_t)w.top,
        (uint8_t)(bottom >> 8), (uint8_t)bottom, 1
    };
    memcpy(out, encoded, sizeof encoded);
    *bytes = (uint32_t)(w.right - w.left) / 8u * (uint32_t)(w.bottom - w.top);
    return true;
}
