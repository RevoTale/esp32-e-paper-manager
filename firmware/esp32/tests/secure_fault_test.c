#include "secure.h"
#include "mbedtls/md.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

static uint8_t key[32], identity[16], challenge[56], auth[72];
static void host_proof(void) {
    const char label[] = "epaper/client/v1";
    uint8_t message[sizeof label - 1 + 16 + 56 + 32];
    size_t position = sizeof label - 1;
    memcpy(message, label, position);
    memcpy(message + position, identity, 16); position += 16;
    memcpy(message + position, challenge, 56); position += 56;
    memcpy(message + position, auth + 8, 32);
    assert(mbedtls_md_hmac(mbedtls_md_info_from_type(MBEDTLS_MD_SHA256),
        key, sizeof key, message, sizeof message, auth + 40) == 0);
}
static void fixture(void) {
    memset(key, 0x36, sizeof key); memset(identity, 0x7b, sizeof identity);
    assert(ep_challenge(key, identity, 2, 3, challenge));
    memset(auth, 0, sizeof auth); memcpy(auth, "EPWP", 4); auth[4] = 1;
    memset(auth + 8, 0x52, 32); host_proof();
}
static void start(ep_secure *s) {
    assert(ep_secure_start(s, key, identity, challenge, auth));
    assert(s->ready && s->send_sequence == 0 && s->receive_sequence == 0);
}
static void handshake_faults(void) {
    ep_secure session;
    fixture(); start(&session); ep_secure_free(&session);
    assert(ep_filled((const uint8_t *)&session, sizeof session, 0));
    for (unsigned byte = 0; byte < sizeof auth; byte++) {
        auth[byte] ^= 1;
        assert(!ep_secure_start(&session, key, identity, challenge, auth));
        assert(!session.ready); ep_secure_free(&session); auth[byte] ^= 1;
    }
    for (unsigned byte = 0; byte < sizeof challenge; byte++) {
        challenge[byte] ^= 1;
        assert(!ep_secure_start(&session, key, identity, challenge, auth));
        assert(!session.ready); ep_secure_free(&session); challenge[byte] ^= 1;
    }
    uint8_t prior[72]; memcpy(prior, auth, sizeof prior);
    assert(ep_challenge(key, identity, 2, 4, challenge));
    assert(!ep_secure_start(&session, key, identity, challenge, prior));
    ep_secure_free(&session);
    fixture(); memset(auth + 8, 0, 32); host_proof(); // Even a valid proof cannot admit zero nonce.
    assert(!ep_secure_start(&session, key, identity, challenge, auth)); ep_secure_free(&session);
    fixture(); key[0] ^= 1;
    assert(!ep_secure_start(&session, key, identity, challenge, auth)); ep_secure_free(&session);
    fixture(); identity[0] ^= 1;
    assert(!ep_secure_start(&session, key, identity, challenge, auth)); ep_secure_free(&session);
    fixture(); assert(!ep_challenge(key, identity, 0, 1, challenge));
    assert(ep_filled(challenge, sizeof challenge, 0));
    assert(!ep_challenge(key, identity, 1, 0, challenge));
    memset(key, 0, sizeof key); assert(!ep_challenge(key, identity, 1, 1, challenge));
    fixture(); memset(identity, 0, sizeof identity); assert(!ep_challenge(key, identity, 1, 1, challenge));
    fixture(); assert(ep_challenge(key, identity, UINT64_MAX, UINT64_MAX, challenge));
    host_proof(); start(&session); ep_secure_free(&session);
}
// Independent record framing using the negotiated receiving key and native GCM.
static size_t host_seal(ep_secure *s, uint64_t sequence, const uint8_t *plain,
                         size_t size, uint8_t *wire) {
    uint8_t nonce[12]; memcpy(nonce, "H2D1", 4); ep_put_be64(nonce + 4, sequence);
    memset(wire, 0, 16); ep_put_be64(wire, sequence);
    wire[8] = (uint8_t)(size >> 8); wire[9] = (uint8_t)size;
    assert(mbedtls_gcm_crypt_and_tag(&s->receive, MBEDTLS_GCM_ENCRYPT, size,
        nonce, sizeof nonce, wire, 16, plain, wire + 16, 16, wire + 16 + size) == 0);
    return size + 32;
}
static void rejected(ep_secure *s, const uint8_t *wire, size_t size, size_t capacity) {
    uint8_t output[1056]; memset(output, 0xa5, sizeof output);
    size_t written = 99; uint64_t sequence = s->receive_sequence;
    assert(!ep_secure_open(s, wire, size, output, capacity, &written));
    assert(written == 0 && s->receive_sequence == sequence);
    assert(ep_filled(output, sizeof output, 0xa5));
}
static void record_faults(void) {
    ep_secure session; uint8_t wire[1089], plain[1057], output[1056];
    memset(plain, 0x81, sizeof plain); fixture(); start(&session);
    size_t size = host_seal(&session, 0, plain, 17, wire), written;
    for (size_t byte = 0; byte < size; byte++) {
        wire[byte] ^= 1; rejected(&session, wire, size, sizeof output); wire[byte] ^= 1;
    }
    for (size_t cut = 0; cut < size; cut++) rejected(&session, wire, cut, sizeof output);
    rejected(&session, wire, size + 1, sizeof output);
    rejected(&session, wire, size, 16);
    size = host_seal(&session, 1, plain, 17, wire); rejected(&session, wire, size, sizeof output);
    size = host_seal(&session, 0, plain, 17, wire);
    assert(ep_secure_open(&session, wire, size, output, sizeof output, &written));
    assert(written == 17 && !memcmp(output, plain, written));
    rejected(&session, wire, size, sizeof output); // Prior authenticated envelope replay.
    size = host_seal(&session, 1, plain, 1057, wire); rejected(&session, wire, size, sizeof output);
    ep_secure_free(&session);
}
static void record_boundaries(void) {
    ep_secure session; uint8_t plain[1057], wire[1089], output[1056], nonce[12];
    memset(plain, 0xd2, sizeof plain); fixture(); start(&session);
    const size_t lengths[] = {0, 1056};
    for (unsigned i = 0; i < sizeof lengths / sizeof lengths[0]; i++) {
        size_t size = lengths[i], written = 99;
        size_t envelope = host_seal(&session, session.receive_sequence, plain, size, wire);
        assert(ep_secure_open(&session, wire, envelope, output, size, &written));
        assert(written == size && !memcmp(output, plain, size));
        uint64_t sequence = session.send_sequence;
        assert(ep_secure_seal(&session, plain, size, wire, size + 32, &written));
        assert(written == size + 32 && ep_be64(wire) == sequence);
        assert(ep_filled(wire + 10, 6, 0));
        memcpy(nonce, "D2H1", 4); ep_put_be64(nonce + 4, sequence);
        assert(mbedtls_gcm_auth_decrypt(&session.send, size, nonce, sizeof nonce, wire, 16,
            wire + 16 + size, 16, wire + 16, output) == 0);
        assert(!memcmp(output, plain, size));
    }
    size_t written = 99; uint64_t sequence = session.send_sequence;
    memset(wire, 0xa5, sizeof wire);
    assert(!ep_secure_seal(&session, plain, 1057, wire, sizeof wire, &written));
    assert(written == 0 && session.send_sequence == sequence && ep_filled(wire, sizeof wire, 0xa5));
    assert(!ep_secure_seal(&session, plain, 1056, wire, 1087, &written));
    assert(written == 0 && session.send_sequence == sequence && ep_filled(wire, sizeof wire, 0xa5));
    session.send_sequence = UINT64_MAX - 1;
    assert(ep_secure_seal(&session, plain, 0, wire, 32, &written));
    assert(session.send_sequence == UINT64_MAX && ep_be64(wire) == UINT64_MAX - 1);
    assert(!ep_secure_seal(&session, plain, 0, wire, 32, &written) && written == 0);
    session.receive_sequence = UINT64_MAX - 1;
    size_t envelope = host_seal(&session, UINT64_MAX - 1, plain, 0, wire);
    assert(ep_secure_open(&session, wire, envelope, output, 0, &written));
    assert(session.receive_sequence == UINT64_MAX && written == 0);
    envelope = host_seal(&session, UINT64_MAX, plain, 0, wire);
    rejected(&session, wire, envelope, sizeof output);
    ep_secure_free(&session); rejected(&session, wire, envelope, sizeof output);
    assert(!ep_secure_seal(&session, plain, 0, wire, 32, &written) && written == 0);
}
int main(void) {
    handshake_faults(); record_faults(); record_boundaries();
    puts("secure faults: authentication, replay, tag/plaintext isolation and bounds passed");
}
