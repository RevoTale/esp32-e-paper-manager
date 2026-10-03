#include "wire.h"
#include <assert.h>
#include <string.h>

int main(void) {
    uint8_t bytes[1056] = {0}, payload[1024];
    memset(payload, 0x93, sizeof payload);
    ep_record hello = {.kind = 1}; ep_record decoded;
    assert(ep_wire_encode(bytes, sizeof bytes, &hello) == 32);
    assert(!memcmp(bytes, "EPS2", 4));
    assert(ep_wire_decode(bytes, 32, &decoded) && decoded.kind == 1);
    for (unsigned i = 0; i < 32; i++) {
        bytes[i] ^= 1; assert(!ep_wire_decode(bytes, 32, &decoded)); bytes[i] ^= 1;
    }
    ep_record data = {.kind = 5, .pass = 1, .epoch = 17, .id = 999,
        .offset = 47000, .size = 1000, .payload = payload};
    assert(ep_wire_encode(bytes, sizeof bytes, &data) == 1032);
    assert(ep_wire_decode(bytes, 1032, &decoded));
    assert(decoded.offset == 47000 && decoded.epoch == 17 && decoded.id == 999 && decoded.pass == 1);
    assert(!memcmp(decoded.payload, payload, 1000));
    assert(!ep_wire_decode(bytes, 1031, &decoded));
    assert(!ep_wire_decode(bytes, 1033, &decoded));
    bytes[800] ^= 1; assert(!ep_wire_decode(bytes, 1032, &decoded));
    data.pass = 2; assert(ep_wire_encode(bytes, sizeof bytes, &data) == 0);
    data.pass = 0; data.size = 1025; assert(ep_wire_encode(bytes, sizeof bytes, &data) == 0);
    data.size = 0; assert(ep_wire_encode(bytes, sizeof bytes, &data) == 0);
    assert(ep_crc32("123456789", 9) == UINT32_C(0xcbf43926));
}
