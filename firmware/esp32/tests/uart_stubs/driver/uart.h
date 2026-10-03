#pragma once
#include "esp_err.h"
#include <stddef.h>
#include <stdint.h>
typedef uint32_t TickType_t;
#define pdMS_TO_TICKS(ms) ((TickType_t)(ms))
#define UART_NUM_0 0
#define UART_DATA_8_BITS 8
#define UART_PARITY_DISABLE 0
#define UART_STOP_BITS_1 1
#define UART_HW_FLOWCTRL_DISABLE 0
#define UART_SCLK_DEFAULT 0
#define UART_PIN_NO_CHANGE (-1)
typedef struct {
    int baud_rate, data_bits, parity, stop_bits, flow_ctrl, source_clk;
} uart_config_t;
esp_err_t uart_param_config(int port, const uart_config_t *config);
esp_err_t uart_set_pin(int port, int tx, int rx, int rts, int cts);
esp_err_t uart_driver_install(int port, int rx, int tx, int queue_size,
                              void *queue, int flags);
int uart_read_bytes(int port, void *out, uint32_t size, TickType_t ticks);
int uart_write_bytes(int port, const void *data, size_t size);
esp_err_t uart_wait_tx_done(int port, TickType_t ticks);
esp_err_t uart_flush_input(int port);
