#pragma once
#include <stdbool.h>
#include <stdint.h>
void ep_network_block(bool);
bool ep_network_live(uint64_t connection);
void ep_network_health(uint8_t out[8]);
