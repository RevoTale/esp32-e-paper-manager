#pragma once
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

enum { EP_REGION_SIZE = 148 };
typedef struct {
    uint8_t digest[32], baseline[32], old_digest[32], new_digest[32];
    uint16_t left, top, right, bottom;
    uint32_t bytes, normal_ms, urgent_ms;
    uint8_t priority;
} ep_region;

// Pure EPS2 content validation, before any controller operation. The adapter
// still owns physical limits; this codec neither enables partial nor checks
// live baseline ownership. Failure leaves output unchanged. See docs/eps2-region.md.
bool ep_region_decode(const uint8_t *, size_t, uint16_t width, uint16_t height, ep_region *);
