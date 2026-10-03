#include "screen.h"
#include <string.h>

bool ep_screen_regions(ep_screen *s, ep_region_sink sink) {
    if (s->generation || s->config.passes != 2 || !sink.valid || !sink.begin) return false;
    s->regions = sink;
    return true;
}

uint8_t ep_screen_begin_region(ep_screen *s, const ep_record *r, uint64_t now) {
    if (!s->regions.begin) return 12;
    if (s->fatal) return 11;
    if (s->state == 1 || s->state == 2) return 3;
    if (r->id <= s->consumed) return 4;
    ep_region region;
    if (!ep_region_decode(r->payload, r->size, s->config.width, s->config.height, &region) ||
        !s->regions.valid(s->sink.context, &region)) return 12;
    // The digest names the last confirmed transaction, full or partial. Do not
    // infer a baseline from controller RAM or retained visible pixels on reboot.
    if (!s->current_image || s->state != 3 || memcmp(s->digest, region.baseline, 32)) return 5;
    uint32_t interval = region.priority ? region.urgent_ms : region.normal_ms;
    if (now - s->last_refresh < interval) return 6;
    s->region = region; s->region_mode = true;
    s->consumed = s->transaction = r->id; memcpy(s->digest, region.digest, 32);
    s->state = 1; s->pass = s->failure = 0; s->offset = 0; s->current_image = false;
    s->started = s->progress = now;
    if (mbedtls_sha256_starts(&s->hash, 0) || s->regions.begin(s->sink.context, &s->region)) {
        s->fatal = true; return ep_screen_fail(s, 11);
    }
    return 0;
}
