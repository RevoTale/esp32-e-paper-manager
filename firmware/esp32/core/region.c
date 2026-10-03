#include "region.h"
#include "bytes.h"
#include "mbedtls/sha256.h"
#include <string.h>

static bool digest_matches(const uint8_t *payload) {
    // Include the zero terminator in domain separation, identically to Go.
    static const uint8_t domain[] = "EPS2-region-v1";
    uint8_t input[sizeof domain + EP_REGION_SIZE - 32], digest[32];
    memcpy(input, domain, sizeof domain);
    memcpy(input + sizeof domain, payload + 32, EP_REGION_SIZE - 32);
    return !mbedtls_sha256(input, sizeof input, digest, 0) && !memcmp(digest, payload, 32);
}

bool ep_region_decode(const uint8_t *p, size_t n, uint16_t width, uint16_t height, ep_region *out) {
    if (!p || !out || n != EP_REGION_SIZE || p[136] > 1 || p[137] || p[138] || p[139] ||
        ep_filled(p + 32, 32, 0)) return false;
    ep_region r = {.left = (uint16_t)ep_le(p + 128, 2), .top = (uint16_t)ep_le(p + 130, 2),
        .right = (uint16_t)ep_le(p + 132, 2), .bottom = (uint16_t)ep_le(p + 134, 2),
        .priority = p[136], .normal_ms = (uint32_t)ep_le(p + 140, 4),
        .urgent_ms = (uint32_t)ep_le(p + 144, 4)};
    if (r.left >= r.right || r.top >= r.bottom || r.right > width || r.bottom > height ||
        r.left % 8 || r.right % 8 || !r.normal_ms || !r.urgent_ms || !digest_matches(p)) return false;
    r.bytes = (uint32_t)(r.right - r.left) / 8u * (uint32_t)(r.bottom - r.top);
    memcpy(r.digest, p, 32); memcpy(r.baseline, p + 32, 32);
    memcpy(r.old_digest, p + 64, 32); memcpy(r.new_digest, p + 96, 32);
    *out = r;
    return true;
}
