#pragma once
#include "screen.h"

typedef enum { EP_CS, EP_DC, EP_RESET } ep_panel_pin;
typedef struct {
    void *context;
    int (*spi)(void *, const uint8_t *, size_t);
    int (*pin)(void *, ep_panel_pin, bool);
    int (*busy)(void *); // 1 ready, 0 busy, negative error.
    uint64_t (*now_us)(void *);
    void (*delay_us)(void *, uint32_t);
} ep_panel_io;
typedef struct {
    ep_panel_io io;
    uint32_t offset, plane_bytes;
    uint8_t command, pass;
    bool active, partial;
    uint64_t started_us;
    uint32_t cycle, elapsed_ms, samples[3], low_samples[3];
    uint8_t state, phase, step;
    bool failed;
    uint8_t error_code, busy_flags;
} ep_panel75;
ep_sink ep_panel75_create(ep_panel75 *, ep_panel_io);
// Candidate adapter; board enablement is opt-in until physical qualification.
ep_region_sink ep_panel75_regions(void);
int ep_panel75_start(ep_panel75 *);
int ep_panel_command(ep_panel75 *, uint8_t, const uint8_t *, size_t);
int ep_panel_data(ep_panel75 *, const uint8_t *, size_t);
int ep_panel_ready(ep_panel75 *, uint32_t budget_us);
int ep_panel_sleep(ep_panel75 *);
int ep_panel_error(ep_panel75 *);
void ep_panel_status(const ep_panel75 *, uint8_t out[36]);
void ep_panel_trace_command(ep_panel75 *, uint8_t command);
void ep_panel_diagnostic(const ep_panel75 *, uint8_t out[11]);
