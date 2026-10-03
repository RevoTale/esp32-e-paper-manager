#include "secure.h"
#include "mbedtls/constant_time.h"
#include "mbedtls/md.h"
#include "mbedtls/platform_util.h"
#include <string.h>

// Exact labels and concatenation from securetransport/session.go, not a new KDF.
static bool mac(const uint8_t key[32], const char *label, const uint8_t *a, size_t na,
                const uint8_t *b, size_t nb, const uint8_t *c, size_t nc, uint8_t out[32]) {
    uint8_t message[256]; size_t size = strlen(label);
    if (size > sizeof message || na > sizeof message-size || nb > sizeof message-size-na ||
        nc > sizeof message-size-na-nb) return false;
    memcpy(message, label, size);
    if (na) memcpy(message + size, a, na);
    size += na;
    if (nb) memcpy(message + size, b, nb);
    size += nb;
    if (nc) memcpy(message + size, c, nc);
    size += nc;
    int result = mbedtls_md_hmac(mbedtls_md_info_from_type(MBEDTLS_MD_SHA256), key, 32, message, size, out);
    mbedtls_platform_zeroize(message, sizeof message);
    return result == 0;
}
bool ep_challenge(const uint8_t key[32], const uint8_t id[16], uint64_t epoch,
                  uint64_t number, uint8_t challenge[56]) {
    memset(challenge, 0, 56);
    if (!epoch || !number || ep_filled(key, 32, 0) || ep_filled(id, 16, 0)) return false;
    memcpy(challenge, "EPWA", 4); challenge[4] = 1;
    ep_put_be64(challenge + 8, epoch); ep_put_be64(challenge + 16, number);
    return mac(key, "epaper/server/v1", id, 16, challenge, 24, NULL, 0, challenge + 24);
}
void ep_secure_free(ep_secure *s) {
    mbedtls_gcm_free(&s->send); mbedtls_gcm_free(&s->receive);
    mbedtls_platform_zeroize(s, sizeof *s);
}
bool ep_secure_start(ep_secure *s, const uint8_t key[32], const uint8_t id[16],
                     const uint8_t challenge[56], const uint8_t auth[72]) {
    memset(s, 0, sizeof *s);
    mbedtls_gcm_init(&s->send); mbedtls_gcm_init(&s->receive);
    uint8_t expected[56], proof[32], master[32], host_key[32], device_key[32];
    bool ok = ep_challenge(key, id, ep_be64(challenge + 8), ep_be64(challenge + 16), expected) &&
        mbedtls_ct_memcmp(expected, challenge, 56) == 0 && !memcmp(auth, "EPWP", 4) && auth[4] == 1 &&
        ep_filled(auth + 5, 3, 0) && !ep_filled(auth + 8, 32, 0) &&
        mac(key, "epaper/client/v1", id, 16, challenge, 56, auth + 8, 32, proof) &&
        mbedtls_ct_memcmp(proof, auth + 40, 32) == 0 &&
        mac(key, "epaper/session/v1", id, 16, challenge, 56, auth + 8, 32, master) &&
        mac(master, "epaper/host-to-device/v1", NULL, 0, NULL, 0, NULL, 0, host_key) &&
        mac(master, "epaper/device-to-host/v1", NULL, 0, NULL, 0, NULL, 0, device_key) &&
        mbedtls_gcm_setkey(&s->receive, MBEDTLS_CIPHER_ID_AES, host_key, 256) == 0 &&
        mbedtls_gcm_setkey(&s->send, MBEDTLS_CIPHER_ID_AES, device_key, 256) == 0;
    mbedtls_platform_zeroize(master, sizeof master); mbedtls_platform_zeroize(host_key, sizeof host_key);
    mbedtls_platform_zeroize(device_key, sizeof device_key); mbedtls_platform_zeroize(proof, sizeof proof);
    if (!ok) ep_secure_free(s);
    s->ready = ok;
    return ok;
}
