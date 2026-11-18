#pragma once
#include "provision.h"
// Called during boot, before the USB and network owners start.
void ep_wifi_setup_boot(const ep_flash *, const uint8_t config[512], uint64_t generation);
