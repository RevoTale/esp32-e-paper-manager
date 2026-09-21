#pragma once
#include "bytes.h"

// Callers serialize the entire operation, not merely individual flash calls.
typedef struct {
    void *context;
    int (*read)(void *, size_t, void *, size_t);
    int (*write)(void *, size_t, const void *, size_t);
    int (*erase)(void *, size_t, size_t);
} ep_flash;
typedef enum { EP_STORE_OK, EP_STORE_IO, EP_STORE_CORRUPT, EP_STORE_EXHAUSTED } ep_store_result;
ep_store_result ep_epoch_reserve(const ep_flash *flash, uint64_t *epoch);
