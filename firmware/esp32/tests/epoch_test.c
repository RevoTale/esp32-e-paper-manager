#include "epoch.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

static unsigned char flash[16384];
static int fail_read, fail_write, fail_erase;
static unsigned writes, erases;
static int read_flash(void *context, size_t offset, void *out, size_t size) {
    (void)context;
    assert(offset <= sizeof flash && size <= sizeof flash - offset);
    if (fail_read) return -1;
    memcpy(out, flash + offset, size);
    return 0;
}
static int write_flash(void *context, size_t offset, const void *in, size_t size) {
    (void)context;
    const unsigned char *source = in;
    assert(offset % 32 == 0 && size == 32 && offset >= 8192);
    writes++;
    for (size_t i = 0; i < size; i++) {
        if (fail_write && i == 8) return -1;
        assert((flash[offset + i] & source[i]) == source[i]);
        flash[offset + i] &= source[i];
    }
    return 0;
}
static int erase_flash(void *context, size_t offset, size_t size) {
    (void)context;
    assert((offset == 8192 || offset == 12288) && size == 4096);
    erases++;
    if (fail_erase) return -1;
    memset(flash + offset, 255, size);
    return 0;
}
static const ep_flash io = {NULL, read_flash, write_flash, erase_flash};
static void fresh(void) {
    memset(flash, 255, sizeof flash);
    fail_read = fail_write = fail_erase = 0;
    writes = erases = 0;
}
int main(void) {
    uint64_t epoch = 99;
    fresh();
    for (uint64_t want = 1; want <= 520; want++) {
        assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_OK);
        assert(epoch == want);
    }
    assert(erases == 4 && writes == 520);
    for (size_t i = 0; i < 8192; i++) assert(flash[i] == 255);
    fresh(); fail_read = 1;
    assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_IO && epoch == 0);
    assert(writes == 0 && erases == 0);
    fresh(); fail_write = 1;
    assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_IO && epoch == 0);
    fail_write = 0;
    assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_CORRUPT && epoch == 0);
    assert(writes == 1 && erases == 0);
    fresh();
    for (unsigned i = 0; i < 128; i++) assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_OK);
    fail_erase = 1;
    assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_IO && epoch == 0);
    fail_erase = 0;
    assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_OK && epoch == 129);
    puts("epoch: reboot monotonicity, rollover, torn write and I/O failure passed");
}
