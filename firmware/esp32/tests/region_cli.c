#include "region.h"
#include "bytes.h"
#include <stdio.h>
#include <string.h>

static void report(const ep_region *r, uint8_t out[121]) {
    memcpy(out + 1, r->baseline, 32);
    memcpy(out + 33, r->old_digest, 32);
    memcpy(out + 65, r->new_digest, 32);
    ep_put_le(out + 97, r->left, 2); ep_put_le(out + 99, r->top, 2);
    ep_put_le(out + 101, r->right, 2); ep_put_le(out + 103, r->bottom, 2);
    out[105] = r->priority;
    ep_put_le(out + 109, r->normal_ms, 4); ep_put_le(out + 113, r->urgent_ms, 4);
    ep_put_le(out + 117, r->bytes, 4);
}

int main(void) {
    uint8_t input[4 + EP_REGION_SIZE];
    size_t count;
    while ((count = fread(input, 1, sizeof input, stdin)) != 0) {
        if (count != sizeof input) return 1;
        ep_region decoded;
        uint8_t out[121] = {0};
        if (ep_region_decode(input + 4, EP_REGION_SIZE, (uint16_t)ep_le(input, 2),
                             (uint16_t)ep_le(input + 2, 2), &decoded)) report(&decoded, out);
        else out[0] = 1;
        if (fwrite(out, 1, sizeof out, stdout) != sizeof out) return 2;
    }
    return ferror(stdin) || fflush(stdout) ? 3 : 0;
}
