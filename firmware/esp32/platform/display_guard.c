#include "display_guard.h"
#include "esp_attr.h"
#include "esp_system.h"

// RTC retention survives software/watchdog reset, not removal of board power.
// https://docs.espressif.com/projects/esp-idf/en/v5.5.5/esp32/api-guides/memory-types.html
static RTC_NOINIT_ATTR uint32_t rail_cookie[2];
enum { COOKIE = 0x45505246 };
static ep_sink target;
static ep_region_sink region_target;
bool ep_display_recovery_required(void) {
    if (esp_reset_reason() == ESP_RST_POWERON) { rail_cookie[0] = rail_cookie[1] = 0; return false; }
    return rail_cookie[0] == COOKIE || rail_cookie[1] == (uint32_t)~COOKIE;
}
static int begin(void *context) {
    (void)context;
    rail_cookie[0] = COOKIE; rail_cookie[1] = (uint32_t)~COOKIE;
    return target.begin(target.context);
}
static int write_pixels(void *context, uint8_t pass, uint32_t offset, const uint8_t *bytes, size_t n) {
    (void)context; return target.write(target.context, pass, offset, bytes, n);
}
static int finish(bool aborting) {
    int result = aborting ? target.abort(target.context) : target.commit(target.context);
    if (!result) rail_cookie[0] = rail_cookie[1] = 0;
    return result;
}
static int commit(void *context) { (void)context; return finish(false); }
static int abort_frame(void *context) { (void)context; return finish(true); }
ep_sink ep_display_guard(ep_sink sink) {
    target = sink;
    region_target = (ep_region_sink){0};
    return (ep_sink){NULL, begin, write_pixels, commit, abort_frame};
}
static bool valid_region(void *context, const ep_region *r) {
    (void)context;
    return region_target.valid(target.context, r);
}
static int begin_region(void *context, const ep_region *r) {
    (void)context;
    rail_cookie[0] = COOKIE; rail_cookie[1] = (uint32_t)~COOKIE;
    return region_target.begin(target.context, r);
}
ep_region_sink ep_display_guard_regions(ep_region_sink sink) {
    if (!sink.valid || !sink.begin) return (ep_region_sink){0};
    region_target = sink;
    return (ep_region_sink){valid_region, begin_region};
}
