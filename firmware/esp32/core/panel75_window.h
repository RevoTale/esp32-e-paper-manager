#pragma once
#include <stdbool.h>
#include <stdint.h>

// Half-open bounds in physical pixels. Sender supplies final byte-aligned pixels;
// receiver must reject, not round, a rectangle whose data would change meaning.
typedef struct {
    uint16_t left, top, right, bottom;
} ep_panel75_window;

// Produces command0x90's nine data bytes and the packed single-plane byte count.
// Minimum16x2: inclusive end bank/line must exceed the start bank/line (R90h).
// Failure leaves outputs unchanged. This pure helper performs no controller I/O.
bool ep_panel75_encode_window(ep_panel75_window, uint8_t out[9], uint32_t *bytes);
