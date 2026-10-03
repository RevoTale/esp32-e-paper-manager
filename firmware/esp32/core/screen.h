#pragma once
#include "wire.h"
#include "region.h"
#include "mbedtls/sha256.h"

// Single owner. Sink writes borrow pixels; no retained MCU frame buffer.
typedef struct {
    void *context;
    int (*begin)(void *);
    int (*write)(void *, uint8_t, uint32_t, const uint8_t *, size_t);
    int (*commit)(void *);
    int (*abort)(void *);
} ep_sink;
// Optional, configured before acquiring a lease. valid must have no I/O or
// state changes; begin uses the same context/write/commit/abort as ep_sink.
typedef struct {
    bool (*valid)(void *, const ep_region *);
    int (*begin)(void *, const ep_region *);
} ep_region_sink;
typedef struct {
    uint16_t width, height, max_chunk, version;
    uint8_t passes;
    uint32_t profile, minimum_full_ms;
} ep_screen_config;
typedef struct {
    ep_sink sink;
    ep_screen_config config;
    ep_region_sink regions;
    ep_region region;
    mbedtls_sha256_context hash;
    uint8_t boot[16], device_id[16], claim[16], digest[32];
    uint64_t generation, consumed, transaction;
    uint64_t observed, started, progress, last_refresh;
    uint32_t offset;
    uint8_t state, pass, failure;
    bool bound, current_image, fatal, refresh_pending, region_mode;
} ep_screen;
bool ep_screen_init(ep_screen *, ep_screen_config, ep_sink, const uint8_t boot[16], const uint8_t id[16]);
bool ep_screen_regions(ep_screen *, ep_region_sink);
void ep_screen_disconnect(ep_screen *);
uint8_t ep_screen_tick(ep_screen *, uint64_t now_ms);
void ep_screen_completed(ep_screen *, uint8_t operation, uint8_t code,
                         uint64_t entered_ms, uint64_t completed_ms);
// Only fully validated EPS2 records from ONE physical/authenticated connection.
// The transport owner must disconnect before switching and discard stale queued
// connection generations. A bind claim never authorizes concurrent writers.
size_t ep_screen_handle(ep_screen *, const ep_record *, uint64_t now_ms, uint8_t *, size_t);
// Internal lifecycle helpers shared by protocol dispatch and bounded polling.
uint8_t ep_screen_fail(ep_screen *, uint8_t code);
uint8_t ep_screen_frame(ep_screen *, const ep_record *, uint64_t now_ms);
uint8_t ep_screen_begin_region(ep_screen *, const ep_record *, uint64_t now_ms);
size_t ep_screen_reply(const ep_screen *, const ep_record *, uint8_t code, uint8_t *, size_t);
