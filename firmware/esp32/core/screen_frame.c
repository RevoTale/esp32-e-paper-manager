#include "screen.h"
#include <string.h>

static uint8_t begin(ep_screen *s, const ep_record *r, uint64_t now) {
    if (s->fatal) return 11;
    if (s->state == 1 || s->state == 2) return 3;
    if (r->id <= s->consumed) return 4;
    uint32_t interval = s->config.minimum_full_ms;
    if (r->kind == 13) {
        // Authenticated operator cadence, not a waveform/safety override.
        // The adapter remains full-only. Reject partial before touching RAM.
        if (r->payload[32] > 1 || (r->payload[33] != 0 && r->payload[33] != 2) ||
            r->payload[34] || r->payload[35] || !ep_le(r->payload + 36, 4) ||
            !ep_le(r->payload + 40, 4)) return 12;
        interval = (uint32_t)ep_le(r->payload + (r->payload[32] ? 40 : 36), 4);
    }
    if (now - s->last_refresh < interval) return 6;
    s->region_mode = false;
    s->consumed = s->transaction = r->id; memcpy(s->digest, r->payload, 32);
    s->state = 1; s->pass = s->failure = 0; s->offset = 0; s->current_image = false;
    s->started = s->progress = now;
    if (mbedtls_sha256_starts(&s->hash, 0) || s->sink.begin(s->sink.context)) {
        s->fatal = true; return ep_screen_fail(s, 11);
    }
    return 0;
}
static uint8_t data(ep_screen *s, const ep_record *r, uint64_t now) {
    if (r->id != s->transaction) return 7;
    if (s->state != 1) return s->state == 2 ? ep_screen_fail(s, 7) : 7;
    uint32_t width = s->region_mode ? (uint32_t)(s->region.right - s->region.left) : s->config.width;
    uint32_t stride = (width + 7) / 8;
    uint32_t frame_bytes = s->region_mode ? s->region.bytes : stride * s->config.height;
    if (r->kind != 5 || r->pass != s->pass || r->offset != s->offset || !r->size ||
        r->size > s->config.max_chunk || r->size > frame_bytes - s->offset) return ep_screen_fail(s, 8);
    uint8_t padding = (uint8_t)(width % 8 ? (1u << (8 - width % 8)) - 1 : 0);
    for (uint32_t i = 0; padding && i < r->size; i++)
        if ((r->offset + i + 1) % stride == 0 && (r->payload[i] & padding)) return ep_screen_fail(s, 8);
    if (s->sink.write(s->sink.context, r->pass, r->offset, r->payload, r->size) ||
        mbedtls_sha256_update(&s->hash, r->payload, r->size)) {
        s->fatal = true; return ep_screen_fail(s, 11);
    }
    s->offset += r->size; s->progress = now;
    if (s->offset == frame_bytes) {
        uint8_t digest[32];
        if (mbedtls_sha256_finish(&s->hash, digest)) return ep_screen_fail(s, 11);
        const uint8_t *expected = s->region_mode ?
            (s->pass ? s->region.new_digest : s->region.old_digest) : s->digest;
        if (memcmp(digest, expected, 32)) return ep_screen_fail(s, 9);
        s->pass++; s->offset = 0;
        if (s->pass == s->config.passes) s->state = 2;
        else if (mbedtls_sha256_starts(&s->hash, 0)) return ep_screen_fail(s, 11);
    }
    return 0;
}
uint8_t ep_screen_frame(ep_screen *s, const ep_record *r, uint64_t now) {
    if (r->kind == 14) return ep_screen_begin_region(s, r, now);
    if (r->kind == 8) {
        if (s->state == 1 || s->state == 2) { ep_screen_fail(s, 0); s->state = 5; }
        return s->fatal ? 11 : 0;
    }
    if (r->kind == 4 || r->kind == 13) return begin(s, r, now);
    if (r->kind == 5 || r->kind == 11) return data(s, r, now);
    if (r->kind != 6 && r->kind != 7) return 1;
    if (r->id != s->transaction) return r->id <= s->consumed ? 4 : (r->kind == 7 ? 0 : 7);
    if (memcmp(r->payload, s->digest, 32)) return 5;
    if (r->kind == 7) return s->failure;
    if (s->state == 3 && s->current_image) return 0;
    if (s->state != 2) return 7;
    s->current_image = false; s->last_refresh = now; s->refresh_pending = true;
    if (s->sink.commit(s->sink.context)) { s->fatal = true; return ep_screen_fail(s, 11); }
    s->state = 3; s->current_image = true;
    return 0;
}
