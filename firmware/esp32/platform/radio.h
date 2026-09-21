#pragma once
#include <stdbool.h>
#include <stdint.h>
bool ep_radio_open(const uint8_t config[512]);
bool ep_radio_join(void);
bool ep_radio_online(void);
