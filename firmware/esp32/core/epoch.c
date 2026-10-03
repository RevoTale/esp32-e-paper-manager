#include "epoch.h"
#include <string.h>

enum { START = 8192, SECTOR = 4096, RECORD = 32, SLOTS = SECTOR / RECORD };
static const uint8_t marker[] = {0xc3, 0x5a, 0x69, 0x96};
typedef struct { uint64_t epoch; int block, slot, free[2]; } scan_result;

static uint64_t decode(const uint8_t p[RECORD]) {
    if (memcmp(p, "EPE1", 4) || memcmp(p + 24, marker, 4) ||
        ep_crc32(p, 20) != ep_be32(p + 20) || !ep_filled(p + 28, 4, 255)) return 0;
    uint64_t epoch = ep_be64(p + 4);
    return ep_be64(p + 12) == ~epoch ? epoch : 0;
}
static ep_store_result scan(const ep_flash *flash, scan_result *s) {
    uint8_t page[RECORD];
    for (int block = 0; block < 2; block++) {
        for (int slot = 0; slot < SLOTS; slot++) {
            size_t offset = START + (size_t)block * SECTOR + (size_t)slot * RECORD;
            if (flash->read(flash->context, offset, page, sizeof page)) return EP_STORE_IO;
            if (ep_filled(page, sizeof page, 255)) {
                if (s->free[block] < 0) s->free[block] = slot;
                continue;
            }
            // Match Go epochstore: any damaged occupied slot is ambiguous.
            // Never fall back to an earlier epoch or automatically erase it.
            uint64_t epoch = decode(page);
            if (!epoch) return EP_STORE_CORRUPT;
            if (epoch > s->epoch) { s->epoch = epoch; s->block = block; s->slot = slot; }
        }
    }
    return EP_STORE_OK;
}
static ep_store_result write_verified(const ep_flash *flash, size_t offset, uint64_t next) {
    uint8_t page[RECORD], verify[RECORD];
    memset(page, 255, sizeof page);
    memcpy(page, "EPE1", 4);
    ep_put_be64(page + 4, next);
    ep_put_be64(page + 12, ~next);
    ep_put_be32(page + 20, ep_crc32(page, 20));
    memcpy(page + 24, marker, sizeof marker);
    if (flash->write(flash->context, offset, page, sizeof page) ||
        flash->read(flash->context, offset, verify, sizeof verify)) return EP_STORE_IO;
    return memcmp(page, verify, sizeof page) == 0 ? EP_STORE_OK : EP_STORE_CORRUPT;
}
ep_store_result ep_epoch_reserve(const ep_flash *flash, uint64_t *epoch) {
    *epoch = 0;
    scan_result s = {0, -1, -1, {-1, -1}};
    ep_store_result result = scan(flash, &s);
    if (result != EP_STORE_OK) return result;
    if (s.epoch == UINT64_MAX) return EP_STORE_EXHAUSTED;
    size_t offset = START;
    if (s.epoch != 0) {
        if (s.free[s.block] > s.slot) {
            offset += (size_t)s.block * SECTOR + (size_t)s.free[s.block] * RECORD;
        } else {
            offset += (size_t)(1 - s.block) * SECTOR;
            if (flash->erase(flash->context, offset, SECTOR)) return EP_STORE_IO;
        }
    }
    result = write_verified(flash, offset, s.epoch + 1);
    if (result == EP_STORE_OK) *epoch = s.epoch + 1;
    return result;
}
