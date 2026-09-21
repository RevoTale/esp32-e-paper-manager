#include "epoch.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

static unsigned char data[16384];
static unsigned reads, writes, erases, read_fail, torn, erase_cut;
static int corrupt;

static int read_flash(void *context, size_t offset, void *out, size_t size) {
    (void)context;
    assert(offset <= sizeof data && size <= sizeof data - offset);
    if (++reads == read_fail) return -1;
    memcpy(out, data + offset, size);
    return 0;
}

static int write_flash(void *context, size_t offset, const void *in, size_t size) {
    (void)context;
    const unsigned char *source = in;
    assert(offset >= 8192 && offset % 32 == 0 && size == 32);
    assert(offset <= sizeof data && size <= sizeof data - offset);
    writes++;
    for (size_t i = 0; i < size; i++) {
        if (torn && i == torn) return -1;
        assert((data[offset + i] & source[i]) == source[i]);
        data[offset + i] &= source[i];
    }
    if (corrupt) data[offset + 20] ^= 1;
    return 0;
}

static int erase_flash(void *context, size_t offset, size_t size) {
    (void)context;
    assert((offset == 8192 || offset == 12288) && size == 4096);
    erases++;
    if (erase_cut) {
        assert(erase_cut < size);
        memset(data + offset, 255, erase_cut);
        return -1;
    }
    memset(data + offset, 255, size);
    return 0;
}

static const ep_flash io = {NULL, read_flash, write_flash, erase_flash};

static void fresh(void) {
    memset(data, 255, sizeof data);
    reads = writes = erases = read_fail = torn = erase_cut = 0;
    corrupt = 0;
}

static void reserve_through(unsigned count) {
    uint64_t epoch;
    for (unsigned want = 1; want <= count; want++) {
        assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_OK);
        assert(epoch == want);
    }
}

static void committed_corruption(void) {
    for (unsigned byte = 0; byte < 32; byte++) {
        uint64_t epoch = 99;
        fresh();
        reserve_through(2);
        data[8192 + 32 + byte] ^= 1;
        assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_CORRUPT && epoch == 0);
        assert(writes == 2 && erases == 0);
    }
}

static void canonical_record_boundaries(void) {
    uint64_t epoch;
    fresh();
    reserve_through(1);
    ep_put_be64(data + 8196, UINT64_MAX);
    ep_put_be64(data + 8204, 0);
    ep_put_be32(data + 8212, ep_crc32(data + 8192, 20));
    assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_EXHAUSTED && epoch == 0);
    assert(writes == 1 && erases == 0);

    for (unsigned invalid = 0; invalid < 2; invalid++) {
        fresh();
        reserve_through(1);
        if (invalid == 0) {
            ep_put_be64(data + 8196, 0);
            ep_put_be64(data + 8204, UINT64_MAX);
        } else {
            ep_put_be64(data + 8204, 1); // Incorrect inverse, valid CRC.
        }
        ep_put_be32(data + 8212, ep_crc32(data + 8192, 20));
        assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_CORRUPT && epoch == 0);
        assert(writes == 1 && erases == 0);
    }
}

static void verification_failures(void) {
    uint64_t epoch;
    fresh();
    read_fail = 257; // Two sectors scanned, then the committed write is read.
    assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_IO && epoch == 0);
    read_fail = 0;
    assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_OK && epoch == 2);

    fresh();
    corrupt = 1;
    assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_CORRUPT && epoch == 0);
    corrupt = 0;
    assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_CORRUPT && epoch == 0);
    assert(writes == 1 && erases == 0);
}

static void torn_rollover(void) {
    for (unsigned cut = 1; cut < 32; cut++) {
        uint64_t epoch;
        fresh();
        reserve_through(128);
        torn = cut;
        assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_IO && epoch == 0);
        torn = 0;
        if (cut < 28) {
            assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_CORRUPT && epoch == 0);
            assert(writes == 129);
        } else {
            // Remaining bytes are erased padding: epoch 129 was durable.
            assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_OK && epoch == 130);
            assert(writes == 130);
        }
        assert(erases == 1);
    }
}

static void interrupted_erase(void) {
    const unsigned cuts[] = {1, 20, 31, 32, 4095};
    for (unsigned i = 0; i < sizeof cuts / sizeof cuts[0]; i++) {
        uint64_t epoch;
        unsigned char newest[32];
        fresh();
        reserve_through(256);
        memcpy(newest, data + 16352, sizeof newest);
        erase_cut = cuts[i];
        assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_IO && epoch == 0);
        assert(memcmp(newest, data + 16352, sizeof newest) == 0);
        erase_cut = 0;
        if (cuts[i] >= 28) {
            // Whole valid records or only already-erased padding were removed.
            assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_OK && epoch == 257);
            assert(writes == 257 && erases == 3);
        } else {
            assert(ep_epoch_reserve(&io, &epoch) == EP_STORE_CORRUPT && epoch == 0);
            assert(writes == 256 && erases == 2);
        }
    }
}

int main(void) {
    committed_corruption();
    canonical_record_boundaries();
    verification_failures();
    torn_rollover();
    interrupted_erase();
    puts("epoch faults: corruption, exhaustion, verification and interrupted rollover passed");
}
