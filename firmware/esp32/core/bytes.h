#pragma once
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

uint64_t ep_be64(const uint8_t *p);
void ep_put_be64(uint8_t *p, uint64_t value);
uint32_t ep_be32(const uint8_t *p);
void ep_put_be32(uint8_t *p, uint32_t value);
uint32_t ep_crc32(const void *data, size_t size);
uint32_t ep_crc32_extend(uint32_t previous, const void *data, size_t size);
uint64_t ep_le(const uint8_t *p, unsigned size);
void ep_put_le(uint8_t *p, uint64_t value, unsigned size);
bool ep_filled(const uint8_t *p, size_t size, uint8_t value);
