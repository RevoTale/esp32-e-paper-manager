#pragma once
#include "epoch.h"
#include "esp_err.h"

// No formatting. The owner task must serialize provisioning and boot epochs.
esp_err_t ep_storage_open(ep_flash *flash);
