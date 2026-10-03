#include "wire.h"
#include <stdio.h>

int main(void) {
    uint8_t input[EP_RECORD], output[EP_RECORD], length[2];
    ep_record record;
    while (fread(length, 1, 2, stdin) == 2) {
        size_t size = (size_t)length[0] * 256 + length[1];
        if (size > sizeof input || fread(input, 1, size, stdin) != size) return 1;
        if (!ep_wire_decode(input, size, &record)) return 2;
        size_t written = ep_wire_encode(output, sizeof output, &record);
        if (written != size || fwrite(output, 1, size, stdout) != size || fflush(stdout)) return 3;
    }
    return ferror(stdin) ? 4 : 0;
}
