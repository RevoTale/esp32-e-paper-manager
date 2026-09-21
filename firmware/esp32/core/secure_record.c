#include "secure.h"
#include "mbedtls/platform_util.h"
#include <string.h>

bool ep_secure_open(ep_secure *s, const uint8_t *wire, size_t size, uint8_t *plain,
                    size_t capacity, size_t *written) {
    *written = 0;
    if (!s->ready || size < 32 || size > 1088) return false;
    size_t length = (size_t)wire[8] * 256 + wire[9];
    if (length > 1056 || length > capacity || length + 32 != size || !ep_filled(wire + 10, 6, 0) ||
        s->receive_sequence == UINT64_MAX || ep_be64(wire) != s->receive_sequence) return false;
    uint8_t nonce[12], scratch[1056];
    memcpy(nonce, "H2D1", 4); ep_put_be64(nonce + 4, s->receive_sequence);
    int result = mbedtls_gcm_auth_decrypt(&s->receive, length, nonce, 12, wire, 16,
        wire + 16 + length, 16, wire + 16, scratch);
    if (!result) { memcpy(plain, scratch, length); *written = length; s->receive_sequence++; }
    mbedtls_platform_zeroize(scratch, sizeof scratch);
    return result == 0;
}
bool ep_secure_seal(ep_secure *s, const uint8_t *plain, size_t size, uint8_t *wire,
                    size_t capacity, size_t *written) {
    *written = 0;
    if (!s->ready || size > 1056 || capacity < size + 32 || s->send_sequence == UINT64_MAX) return false;
    uint8_t nonce[12]; memcpy(nonce, "D2H1", 4); ep_put_be64(nonce + 4, s->send_sequence);
    memset(wire, 0, 16); ep_put_be64(wire, s->send_sequence);
    wire[8] = (uint8_t)(size >> 8); wire[9] = (uint8_t)size;
    // Consume before encrypting. Even a failed seal must not retry this nonce.
    s->send_sequence++;
    int result = mbedtls_gcm_crypt_and_tag(&s->send, MBEDTLS_GCM_ENCRYPT, size, nonce, 12,
        wire, 16, plain, wire + 16, 16, wire + 16 + size);
    if (!result) *written = size + 32;
    return result == 0;
}
