#include "bytes.h"

uint64_t ep_be64(const uint8_t *p) {
    uint64_t value = 0;
    for (unsigned i = 0; i < 8; i++) value = (value << 8) | p[i];
    return value;
}
void ep_put_be64(uint8_t *p, uint64_t value) {
    for (unsigned i = 0; i < 8; i++) { p[7 - i] = (uint8_t)value; value >>= 8; }
}
uint32_t ep_be32(const uint8_t *p) {
    return ((uint32_t)p[0] << 24) | ((uint32_t)p[1] << 16) | ((uint32_t)p[2] << 8) | p[3];
}
void ep_put_be32(uint8_t *p, uint32_t value) {
    for (unsigned i = 0; i < 4; i++) { p[3 - i] = (uint8_t)value; value >>= 8; }
}
uint32_t ep_crc32(const void *data, size_t size) {
    return ep_crc32_extend(0, data, size);
}
uint32_t ep_crc32_extend(uint32_t previous, const void *data, size_t size) {
    const uint8_t *p = data;
    uint32_t crc = ~previous;
    for (size_t i = 0; i < size; i++) {
        crc ^= p[i];
        for (unsigned bit = 0; bit < 8; bit++)
            crc = (crc >> 1) ^ ((crc & 1) ? UINT32_C(0xedb88320) : 0);
    }
    return ~crc;
}
uint64_t ep_le(const uint8_t *p, unsigned size) {
    uint64_t value = 0;
    for (unsigned i = 0; i < size; i++) value |= (uint64_t)p[i] << (8 * i);
    return value;
}
void ep_put_le(uint8_t *p, uint64_t value, unsigned size) {
    for (unsigned i = 0; i < size; i++) { p[i] = (uint8_t)value; value >>= 8; }
}
bool ep_filled(const uint8_t *p, size_t size, uint8_t value) {
    for (size_t i = 0; i < size; i++) if (p[i] != value) return false;
    return true;
}
