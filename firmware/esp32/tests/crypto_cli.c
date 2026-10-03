#include "secure.h"
#include <stdio.h>
#include <string.h>

int main(void) {
    uint8_t key[32], id[16] = "pico-epaper-0001", challenge[56], auth[72];
    for (unsigned i = 0; i < 32; i++) key[i] = (uint8_t)i;
    if (!ep_challenge(key, id, 7, 3, challenge)) return 1;
    if (fwrite(challenge, 1, 56, stdout) != 56 || fflush(stdout)) return 2;
    if (fread(auth, 1, 72, stdin) != 72) return 3;
    ep_secure session;
    bool valid = ep_secure_start(&session, key, id, challenge, auth);
    if (putchar(valid ? 1 : 0) == EOF || fflush(stdout)) return 4;
    if (!valid) return 0;
    uint8_t envelope[1088], plain[1056], output[1088], length[2];
    while (fread(length, 1, 2, stdin) == 2) {
        size_t size = (size_t)length[0] * 256 + length[1];
        if (!size) break;
        if (size > sizeof envelope || fread(envelope, 1, size, stdin) != size) return 5;
        size_t count = 0, sealed = 0;
        valid = ep_secure_open(&session, envelope, size, plain, sizeof plain, &count);
        if (putchar(valid ? 1 : 0) == EOF) return 6;
        if (!valid) { fflush(stdout); break; }
        if (!ep_secure_seal(&session, plain, count, output, sizeof output, &sealed)) return 7;
        length[0] = (uint8_t)(sealed >> 8); length[1] = (uint8_t)sealed;
        if (fwrite(length, 1, 2, stdout) != 2 || fwrite(output, 1, sealed, stdout) != sealed || fflush(stdout)) return 8;
    }
    ep_secure_free(&session);
    return 0;
}
