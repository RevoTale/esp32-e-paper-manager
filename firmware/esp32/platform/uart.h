#pragma once
#include "esp_err.h"
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

esp_err_t ep_uart_open(void);
// Bounded idle poll. A consumed byte stays valid after scheduler delay.
bool ep_uart_poll_byte(uint8_t *out);
// Total request deadline, not extended by each received fragment.
bool ep_uart_read(void *out, size_t size, int64_t deadline_us);
bool ep_uart_write(const void *data, size_t size);
esp_err_t ep_uart_discard(void);
