#pragma once
#include "bytes.h"

enum { EP_HEADER = 32, EP_PAYLOAD = 1024, EP_RECORD = 1056 };
typedef struct {
    uint8_t kind, pass;
    uint64_t epoch, id;
    uint32_t offset;
    uint16_t size;
    const uint8_t *payload; // borrowed until dispatch returns
} ep_record;
// Zero means invalid shape; never use an unchecked peer length for allocation.
size_t ep_wire_size(const uint8_t header[EP_HEADER]);
bool ep_wire_decode(const uint8_t *wire, size_t size, ep_record *record);
size_t ep_wire_encode(uint8_t *wire, size_t capacity, const ep_record *record);
