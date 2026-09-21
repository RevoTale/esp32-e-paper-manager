#pragma once
#include "owner.h"
#include "freertos/FreeRTOS.h"
#include "freertos/queue.h"
typedef struct {
    uint64_t connection;
    uint16_t size;
    uint8_t bytes[EP_RECORD];
} ep_message;
typedef struct { QueueHandle_t requests, replies; } ep_channel;
bool ep_channel_open(ep_channel *);
bool ep_channel_exchange(ep_channel *, const ep_message *, ep_message *);
void ep_channel_close(ep_channel *, uint64_t connection);
