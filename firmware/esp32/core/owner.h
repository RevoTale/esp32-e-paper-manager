#pragma once
#include "screen.h"
typedef enum { EP_NONE, EP_USB, EP_NETWORK } ep_source;
typedef struct {
    ep_screen *screen;
    ep_source source;
    uint64_t connection, last_usb, observed, highest[3];
} ep_owner;
// Called serially, never during another sink operation. USB preempts staging,
// not the synchronous physical refresh. Connection IDs are monotonic per source.
bool ep_owner_admit(ep_owner *, ep_source, uint64_t connection, uint64_t now_ms);
void ep_owner_closed(ep_owner *, ep_source, uint64_t connection);
void ep_owner_tick(ep_owner *, uint64_t now_ms);
void ep_owner_completed(ep_owner *, uint64_t now_ms);
