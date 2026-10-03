#include "screen.h"
#include <string.h>

size_t ep_screen_reply(const ep_screen *s, const ep_record *r, uint8_t code, uint8_t *out, size_t capacity) {
    uint8_t body[88] = {0};
    body[0] = r->kind; body[1] = code;
    // Discovery/lease replies describe the lease, not an unrelated transaction.
    if (((r->kind >= 4 && r->kind <= 8) || r->kind == 11 || r->kind == 13 || r->kind == 14) &&
        (r->id == s->transaction || r->kind == 8) && code != 5 && code != 2) {
        body[2] = s->state; body[3] = s->pass; ep_put_le(body + 4, s->offset, 4);
        body[8] = s->current_image && s->state == 3 ? 1 : 0;
    }
    uint64_t elapsed = s->observed - s->last_refresh;
    uint32_t interval = s->config.minimum_full_ms;
    if (r->kind == 13 && r->size == 44 && r->payload[32] <= 1)
        interval = (uint32_t)ep_le(r->payload + (r->payload[32] ? 40 : 36), 4);
    if (r->kind == 14 && r->size == 148 && r->payload[136] <= 1)
        interval = (uint32_t)ep_le(r->payload + (r->payload[136] ? 144 : 140), 4);
    ep_put_le(body + 20, elapsed < interval ? interval - elapsed : 0, 4);
    ep_put_le(body + 24, s->generation, 8); memcpy(body + 32, s->boot, 16);
    uint16_t size = 48;
    if (r->kind == 1) {
        uint8_t *caps = body + 48;
        ep_put_le(caps, s->config.width, 2); ep_put_le(caps + 2, s->config.height, 2);
        ep_put_le(caps + 4, ((uint32_t)s->config.width + 7) / 8, 2);
        ep_put_le(caps + 6, s->config.max_chunk, 2); caps[8] = s->config.passes; caps[9] = 1;
        ep_put_le(caps + 10, s->regions.begin ? 27 : 11, 2);
        ep_put_le(caps + 12, s->config.profile, 4); ep_put_le(caps + 16, s->config.version, 2);
        ep_put_le(caps + 20, s->config.minimum_full_ms, 4); memcpy(caps + 24, s->device_id, 16); size = 88;
    }
    ep_record reply = {.kind = 9, .epoch = r->epoch, .id = r->id, .size = size, .payload = body};
    return ep_wire_encode(out, capacity, &reply);
}
