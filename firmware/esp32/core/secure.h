#pragma once
#include "bytes.h"
#include "mbedtls/gcm.h"

typedef struct {
    mbedtls_gcm_context send, receive;
    uint64_t send_sequence, receive_sequence;
    bool ready;
} ep_secure;
bool ep_challenge(const uint8_t key[32], const uint8_t id[16], uint64_t epoch,
                  uint64_t number, uint8_t challenge[56]);
// Fresh/freed context only; callers fully buffer both handshake messages first.
bool ep_secure_start(ep_secure *, const uint8_t key[32], const uint8_t id[16],
                     const uint8_t challenge[56], const uint8_t auth[72]);
void ep_secure_free(ep_secure *);
bool ep_secure_open(ep_secure *, const uint8_t *wire, size_t size, uint8_t *plain,
                    size_t capacity, size_t *written);
// Input/output must not overlap. Close the transport after ANY envelope error.
bool ep_secure_seal(ep_secure *, const uint8_t *plain, size_t size, uint8_t *wire,
                    size_t capacity, size_t *written);
