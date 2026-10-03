#include "screen.h"
#include <string.h>

static bool active(const ep_screen *s) { return s->state == 1 || s->state == 2; }
bool ep_screen_init(ep_screen *s, ep_screen_config config, ep_sink sink, const uint8_t boot[16], const uint8_t id[16]) {
    if (!config.width || !config.height || !config.max_chunk || config.max_chunk > 1024 ||
        !config.version || !config.profile || !config.minimum_full_ms || !config.passes || config.passes > 2 ||
        !sink.begin || !sink.write || !sink.commit || !sink.abort || ep_filled(boot, 16, 0)) return false;
    memset(s, 0, sizeof *s); s->sink = sink; s->config = config;
    memcpy(s->boot, boot, 16); memcpy(s->device_id, id, 16);
    mbedtls_sha256_init(&s->hash);
    return true;
}
uint8_t ep_screen_fail(ep_screen *s, uint8_t code) {
    if (code == 11) s->fatal = true;
    if (active(s) && s->sink.abort(s->sink.context)) { s->fatal = true; code = 11; }
    s->state = 4; s->failure = code;
    return code;
}
void ep_screen_disconnect(ep_screen *s) {
    if (active(s)) { ep_screen_fail(s, 0); s->state = 5; }
    s->bound = false;
}
uint8_t ep_screen_tick(ep_screen *s, uint64_t now) {
    if (now < s->observed) return active(s) ? ep_screen_fail(s, 10) : 10;
    s->observed = now;
    if (active(s) && (now - s->progress >= 20000 || now - s->started >= 120000)) return ep_screen_fail(s, 10);
    return 0;
}
static uint8_t acquire(ep_screen *s, const ep_record *r) {
    if (memcmp(s->boot, r->payload, 16) || ep_filled(r->payload + 16, 16, 0) || r->epoch == UINT64_MAX) return 2;
    if (s->generation == r->epoch + 1 && !memcmp(s->claim, r->payload + 16, 16)) return 0;
    if (s->generation != r->epoch) return 2;
    if (s->bound || active(s)) return 3;
    s->generation++; memcpy(s->claim, r->payload + 16, 16);
    s->state = s->pass = s->failure = 0; s->offset = 0;
    s->consumed = s->transaction = 0; s->current_image = false;
    return 0;
}
static uint8_t bind(ep_screen *s, const ep_record *r) {
    if (s->generation != r->epoch || !r->epoch || memcmp(s->boot, r->payload, 16) ||
        ep_filled(s->claim, 16, 0) || memcmp(s->claim, r->payload + 16, 16)) return 2;
    s->bound = true;
    return 0;
}
size_t ep_screen_handle(ep_screen *s, const ep_record *r, uint64_t now, uint8_t *out, size_t capacity) {
    uint8_t code = 0;
    if (r->kind != 10 && r->kind != 12) {
        code = ep_screen_tick(s, now);
        if (code) return ep_screen_reply(s, r, code, out, capacity);
    }
    switch (r->kind) {
    case 1: break;
    case 2: code = acquire(s, r); break;
    case 3: code = bind(s, r); break;
    case 10: case 12: code = 12; break; // no fabricated hardware observation
    default:
        if (!s->bound || r->epoch != s->generation) code = 2;
        else code = ep_screen_frame(s, r, now);
        break;
    }
    return ep_screen_reply(s, r, code, out, capacity);
}
