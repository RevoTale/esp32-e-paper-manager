#include "panel75_window.h"
#include <assert.h>
#include <string.h>

static void check_window(ep_panel75_window w, const uint8_t expected[9], uint32_t bytes) {
    uint8_t encoded[9];
    uint32_t count = 0;
    assert(ep_panel75_encode_window(w, encoded, &count));
    assert(!memcmp(encoded, expected, sizeof encoded));
    assert(count == bytes);
}

static void coordinate_boundaries(void) {
    const uint8_t full[] = {0, 0, 3, 31, 0, 0, 1, 223, 1};
    check_window((ep_panel75_window){0, 0, 800, 480}, full, 48000);
    const uint8_t low_byte_borrow[] = {0, 240, 0, 255, 0, 254, 0, 255, 1};
    check_window((ep_panel75_window){240, 254, 256, 256}, low_byte_borrow, 4);
    const uint8_t second_boundary[] = {1, 240, 1, 255, 0, 255, 1, 0, 1};
    check_window((ep_panel75_window){496, 255, 512, 257}, second_boundary, 4);
    const uint8_t last_pixel_row[] = {3, 16, 3, 31, 1, 222, 1, 223, 1};
    check_window((ep_panel75_window){784, 478, 800, 480}, last_pixel_row, 4);
}

static void invalid_windows_leave_outputs_unchanged(void) {
    const ep_panel75_window invalid[] = {
        {0, 0, 0, 1}, {8, 0, 8, 1}, {16, 0, 8, 1},
        {0, 1, 8, 1}, {0, 2, 8, 1}, {1, 0, 8, 1},
        {0, 0, 7, 1}, {792, 0, 808, 1}, {0, 479, 8, 481},
        {UINT16_MAX, 0, 8, 1}, {0, 0, UINT16_MAX, UINT16_MAX},
        {0, 0, 8, 480}, {0, 0, 800, 1}, {792, 478, 800, 480}
    };
    uint8_t encoded[9], sentinel[9];
    memset(sentinel, 0xa5, sizeof sentinel);
    for (size_t i = 0; i < sizeof invalid / sizeof invalid[0]; i++) {
        memcpy(encoded, sentinel, sizeof encoded);
        uint32_t count = 123;
        assert(!ep_panel75_encode_window(invalid[i], encoded, &count));
        assert(!memcmp(encoded, sentinel, sizeof encoded));
        assert(count == 123);
    }
    ep_panel75_window valid = {0, 0, 16, 2};
    uint32_t count = 123;
    assert(!ep_panel75_encode_window(valid, NULL, &count));
    assert(count == 123);
    assert(!ep_panel75_encode_window(valid, encoded, NULL));
}

static uint16_t coordinate(const uint8_t *p) {
    return (uint16_t)((uint16_t)p[0] * 256u + p[1]);
}

static void aligned_horizontal_windows_round_trip(void) {
    for (uint16_t left = 0; left < 800; left += 8) {
        for (uint16_t right = (uint16_t)(left + 16); right <= 800; right += 8) {
            uint8_t encoded[9];
            uint32_t count;
            assert(ep_panel75_encode_window((ep_panel75_window){left, 0, right, 480}, encoded, &count));
            assert(coordinate(encoded) == left);
            assert(coordinate(encoded + 2) + 1u == right);
            assert(count == (uint32_t)(right - left) / 8u * 480u);
        }
    }
}

int main(void) {
    coordinate_boundaries();
    invalid_windows_leave_outputs_unchanged();
    aligned_horizontal_windows_round_trip();
    return 0;
}
