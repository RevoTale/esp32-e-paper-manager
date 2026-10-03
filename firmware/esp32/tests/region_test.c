#include "region.h"
#include "bytes.h"
#include "mbedtls/sha256.h"
#include <assert.h>
#include <string.h>

static void seal(uint8_t payload[EP_REGION_SIZE]) {
    const uint8_t prefix[] = "EPS2-region-v1";
    uint8_t input[sizeof prefix + EP_REGION_SIZE - 32];
    memcpy(input, prefix, sizeof prefix);
    memcpy(input + sizeof prefix, payload + 32, EP_REGION_SIZE - 32);
    assert(!mbedtls_sha256(input, sizeof input, payload, 0));
}

static void fixture(uint8_t payload[EP_REGION_SIZE]) {
    memset(payload, 0, EP_REGION_SIZE);
    payload[32] = 1; payload[64] = 2; payload[96] = 3;
    ep_put_le(payload + 128, 240, 2); ep_put_le(payload + 130, 254, 2);
    ep_put_le(payload + 132, 256, 2); ep_put_le(payload + 134, 256, 2);
    payload[136] = 1;
    ep_put_le(payload + 140, 1000, 4); ep_put_le(payload + 144, 500, 4);
    seal(payload);
}

static void reject_preserving_output(const uint8_t *p, size_t n, uint16_t w, uint16_t h) {
    ep_region out, original;
    memset(&out, 0xa5, sizeof out); memcpy(&original, &out, sizeof out);
    assert(!ep_region_decode(p, n, w, h, &out));
    assert(!memcmp(&out, &original, sizeof out));
}

static void reject_invalid_semantics(void) {
    const unsigned offsets[] = {32, 128, 132, 136, 137, 138, 139, 140, 144};
    for (size_t i = 0; i < sizeof offsets / sizeof offsets[0]; i++) {
        uint8_t payload[EP_REGION_SIZE]; fixture(payload);
        unsigned at = offsets[i];
        if (at == 32) payload[32] = 0;
        else if (at == 140 || at == 144) memset(payload + at, 0, 4);
        else payload[at] = 255;
        seal(payload); // Semantic rejection, not merely a corrupted hash.
        reject_preserving_output(payload, sizeof payload, 800, 480);
    }
}

int main(void) {
    uint8_t payload[EP_REGION_SIZE], changed[EP_REGION_SIZE]; fixture(payload);
    ep_region out;
    assert(ep_region_decode(payload, sizeof payload, 800, 480, &out));
    assert(out.left == 240 && out.top == 254 && out.right == 256 && out.bottom == 256);
    assert(out.bytes == 4 && out.priority == 1 && out.normal_ms == 1000 && out.urgent_ms == 500);
    assert(!memcmp(out.digest, payload, 32) && out.baseline[0] == 1);
    assert(out.old_digest[0] == 2 && out.new_digest[0] == 3);
    for (size_t i = 0; i < sizeof payload; i++) {
        memcpy(changed, payload, sizeof changed); changed[i] ^= 1;
        reject_preserving_output(changed, sizeof changed, 800, 480);
        reject_preserving_output(payload, i, 800, 480);
    }
    reject_preserving_output(payload, sizeof payload + 1, 800, 480);
    reject_preserving_output(payload, sizeof payload, 255, 480);
    reject_preserving_output(payload, sizeof payload, 800, 255);
    reject_preserving_output(NULL, sizeof payload, 800, 480);
    assert(!ep_region_decode(payload, sizeof payload, 800, 480, NULL));
    reject_invalid_semantics();
}
