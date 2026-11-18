#pragma once
#include "epoch.h"

// Configuration retains the EPC2 wire offsets; only ESP32 auth=2 (WPA2) is admitted.
bool ep_config_valid(const uint8_t config[512]);
bool ep_manager_change_endpoint(const uint8_t *endpoint, size_t size);
enum { EP_SET_MANAGER = 6, EP_INSPECT_MANAGER = 7 };
bool ep_provision_valid(const uint8_t request[512]);
// Used by the network fence and the USB reboot-after-ACK path.
bool ep_provision_may_change(const uint8_t request[512], const uint8_t reply[512]);
// 0 blank, 1 provisioned, 2 corrupt, 3 unreadable. Output cleared on failure.
int ep_config_load(const ep_flash *, uint8_t config[512], uint64_t *generation);
// Physical UART only. Network dispatch must never call this entry point.
// -1 malformed; valid requests receive a canonical, secret-free EPCR reply.
int ep_provision(const ep_flash *, const uint8_t request[512], uint8_t reply[512]);
// Recovery fence: validate original request; never change its operation or flash.
int ep_provision_readonly(const ep_flash *, const uint8_t request[512], uint8_t reply[512]);
bool ep_config_save_wifi(const ep_flash *, const uint8_t candidate[512], uint64_t generation);
