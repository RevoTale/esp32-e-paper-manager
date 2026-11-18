#include "device_tls.h"
#include "network_state.h"
#include "esp_random.h"
#include "esp_timer.h"
#include "lwip/sockets.h"
#include "mbedtls/constant_time.h"
#include "mbedtls/ssl.h"
#include "mbedtls/sha256.h"
#include "psa/crypto.h"
#include <errno.h>
#include <string.h>

// Only the network task owns these objects. The per-device EPN2 exchange starts
// after a successful TLS handshake with the exact USB-enrolled certificate.
static mbedtls_ssl_context client;
static mbedtls_ssl_config config;
static mbedtls_x509_crt anchor;
static uint8_t expected[32];
static bool matched;
static int socket_fd;

static int random_bytes(void *unused, unsigned char *out, size_t len) {
    (void)unused; esp_fill_random(out, len); return 0;
}
static int bio_send(void *ctx, const unsigned char *p, size_t n) {
    int result = send(*(int *)ctx, p, n, 0);
    if (result < 0 && (errno == EWOULDBLOCK || errno == EAGAIN || errno == EINTR)) return MBEDTLS_ERR_SSL_WANT_WRITE;
    return result < 0 ? MBEDTLS_ERR_SSL_INTERNAL_ERROR : result;
}
static int bio_recv(void *ctx, unsigned char *p, size_t n) {
    int result = recv(*(int *)ctx, p, n, 0);
    if (result < 0 && (errno == EWOULDBLOCK || errno == EAGAIN || errno == EINTR)) return MBEDTLS_ERR_SSL_WANT_READ;
    return result < 0 ? MBEDTLS_ERR_SSL_INTERNAL_ERROR : result;
}
static int verify(void *unused, mbedtls_x509_crt *crt, int depth, uint32_t *flags) {
    (void)unused;
    // Enrollment imports a single self-signed leaf, whose exact DER digest is
    // the trust anchor. All other validation flags stay in force.
    uint8_t digest[32];
    if (depth != 0 || mbedtls_sha256(crt->raw.p, crt->raw.len, digest, 0) ||
        mbedtls_ct_memcmp(expected, digest, sizeof digest)) return MBEDTLS_ERR_X509_CERT_VERIFY_FAILED;
    *flags &= ~(uint32_t)MBEDTLS_X509_BADCERT_NOT_TRUSTED;
    matched = *flags == 0;
    return 0;
}
static int nibble(char c) {
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'A' && c <= 'F') return c - 'A' + 10;
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    return -1;
}
static bool wait_tls(int fd, int code, int64_t deadline, uint64_t connection) {
    if (code != MBEDTLS_ERR_SSL_WANT_READ && code != MBEDTLS_ERR_SSL_WANT_WRITE) return false;
    return ep_net_ready(fd, code == MBEDTLS_ERR_SSL_WANT_WRITE, deadline, connection);
}
bool ep_tls_start(int fd, const char *host, const char *hex, uint64_t connection) {
    socket_fd = fd; matched = false;
    mbedtls_ssl_init(&client); mbedtls_ssl_config_init(&config); mbedtls_x509_crt_init(&anchor);
    if (strlen(hex) != 64 || psa_crypto_init() != PSA_SUCCESS) return false;
    for (size_t i = 0; i < sizeof expected; i++) {
        int high = nibble(hex[i*2]), low = nibble(hex[i*2+1]);
        if (high < 0 || low < 0) return false;
        expected[i] = (uint8_t)(high*16 + low);
    }
    if (mbedtls_ssl_config_defaults(&config, MBEDTLS_SSL_IS_CLIENT, MBEDTLS_SSL_TRANSPORT_STREAM, MBEDTLS_SSL_PRESET_DEFAULT)) return false;
    mbedtls_ssl_conf_min_tls_version(&config, MBEDTLS_SSL_VERSION_TLS1_3);
    mbedtls_ssl_conf_max_tls_version(&config, MBEDTLS_SSL_VERSION_TLS1_3);
    mbedtls_ssl_conf_rng(&config, random_bytes, NULL);
    mbedtls_ssl_conf_authmode(&config, MBEDTLS_SSL_VERIFY_REQUIRED);
    mbedtls_ssl_conf_ca_chain(&config, &anchor, NULL);
    mbedtls_ssl_conf_verify(&config, verify, NULL);
    if (mbedtls_ssl_setup(&client, &config) || mbedtls_ssl_set_hostname(&client, host)) return false;
    mbedtls_ssl_set_bio(&client, &socket_fd, bio_send, bio_recv, NULL);
    int result; int64_t deadline = esp_timer_get_time() + 15000000;
    do {
        if (!ep_network_live(connection) || esp_timer_get_time() >= deadline) return false;
        result = mbedtls_ssl_handshake(&client);
    } while (wait_tls(fd, result, deadline, connection));
    return result == 0 && matched && mbedtls_ssl_get_verify_result(&client) == 0;
}
bool ep_tls_transfer(int fd, void *buf, size_t len, int64_t deadline, uint64_t connection, bool writing) {
    uint8_t *p = buf;
    while (len && ep_network_live(connection) && esp_timer_get_time() < deadline) {
        int n = writing ? mbedtls_ssl_write(&client, p, len) : mbedtls_ssl_read(&client, p, len);
        if (n > 0) { p += n; len -= (size_t)n; continue; }
        if (!wait_tls(fd, n, deadline, connection)) return false;
    }
    return !len && ep_network_live(connection);
}
void ep_tls_close(void) {
    mbedtls_ssl_free(&client); mbedtls_ssl_config_free(&config); mbedtls_x509_crt_free(&anchor);
}
