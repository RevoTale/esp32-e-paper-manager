#include "wire.h"
#include <string.h>

static bool shape(const ep_record *r) {
    if (r->size > EP_PAYLOAD) return false;
    if (r->kind == 5 || r->kind == 11) {
        return r->pass < 2 && r->epoch && r->id && r->size >= (r->kind == 11 ? 4 : 1);
    }
    if (r->pass || r->offset) return false;
    switch (r->kind) {
    case 1: case 10: case 12: return !r->epoch && !r->id && !r->size;
    case 2: return !r->id && r->size == 32;
    case 3: return r->epoch && !r->id && r->size == 32;
    case 4: case 6: case 7: return r->epoch && r->id && r->size == 32;
    case 13: return r->epoch && r->id && r->size == 44;
    case 8: return r->epoch && !r->id && !r->size;
    case 9: return r->size == 48 || r->size == 88 || r->size == 56 || r->size == 84;
    default: return false;
    }
}
static ep_record header(const uint8_t *p) {
    return (ep_record){.kind = p[4], .pass = p[5], .size = (uint16_t)ep_le(p + 6, 2),
        .epoch = ep_le(p + 8, 8), .id = ep_le(p + 16, 8), .offset = (uint32_t)ep_le(p + 24, 4)};
}
size_t ep_wire_size(const uint8_t p[EP_HEADER]) {
    if (memcmp(p, "EPS2", 4)) return 0;
    ep_record record = header(p);
    return shape(&record) ? EP_HEADER + (size_t)record.size : 0;
}
bool ep_wire_decode(const uint8_t *wire, size_t size, ep_record *record) {
    if (size < EP_HEADER || size > EP_RECORD || ep_wire_size(wire) != size) return false;
    uint32_t crc = ep_crc32_extend(ep_crc32(wire, 28), wire + EP_HEADER, size - EP_HEADER);
    if (crc != ep_le(wire + 28, 4)) return false;
    *record = header(wire); record->payload = wire + EP_HEADER;
    return true;
}
size_t ep_wire_encode(uint8_t *wire, size_t capacity, const ep_record *r) {
    size_t size = EP_HEADER + (size_t)r->size;
    if (!shape(r) || capacity < size || (r->size && !r->payload)) return 0;
    memset(wire, 0, EP_HEADER); memcpy(wire, "EPS2", 4);
    wire[4] = r->kind; wire[5] = r->pass; ep_put_le(wire + 6, r->size, 2);
    ep_put_le(wire + 8, r->epoch, 8); ep_put_le(wire + 16, r->id, 8);
    ep_put_le(wire + 24, r->offset, 4);
    if (r->size) memcpy(wire + EP_HEADER, r->payload, r->size);
    uint32_t crc = ep_crc32_extend(ep_crc32(wire, 28), wire + EP_HEADER, r->size);
    ep_put_le(wire + 28, crc, 4);
    return size;
}
