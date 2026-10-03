#pragma once
#include "channel.h"
#include "network_state.h"
bool ep_network_start(ep_channel *, const uint8_t config[512], uint64_t epoch);
