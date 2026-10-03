#include "uart.h"
#include "driver/uart.h"
#include "esp_timer.h"

esp_err_t ep_uart_open(void) {
    const uart_config_t config = {.baud_rate = 115200, .data_bits = UART_DATA_8_BITS,
        .parity = UART_PARITY_DISABLE, .stop_bits = UART_STOP_BITS_1,
        .flow_ctrl = UART_HW_FLOWCTRL_DISABLE, .source_clk = UART_SCLK_DEFAULT};
    esp_err_t result = uart_param_config(UART_NUM_0, &config);
    if (result != ESP_OK) return result;
    result = uart_set_pin(UART_NUM_0, 1, 3, UART_PIN_NO_CHANGE, UART_PIN_NO_CHANGE);
    if (result != ESP_OK) return result;
    return uart_driver_install(UART_NUM_0, 4096, 2048, 0, NULL, 0);
}
bool ep_uart_poll_byte(uint8_t *out) {
    TickType_t ticks = pdMS_TO_TICKS(20);
    if (!ticks) ticks = 1;
    // ESP-IDF uart_read_bytes consumes bytes before returning their count.
    // A clock check after success would lose the packet's first byte when the
    // scheduler resumes late. Full-record deadlines remain in ep_uart_read.
    // https://docs.espressif.com/projects/esp-idf/en/v5.5.5/esp32/api-reference/peripherals/uart.html
    return uart_read_bytes(UART_NUM_0, out, 1, ticks) == 1;
}
bool ep_uart_read(void *out, size_t size, int64_t deadline_us) {
    uint8_t *p = out;
    while (size) {
        int64_t remaining = deadline_us - esp_timer_get_time();
        if (remaining <= 0 || size > UINT32_MAX) return false;
        TickType_t ticks = pdMS_TO_TICKS((uint32_t)(remaining / 1000));
        if (!ticks) ticks = 1;
        int count = uart_read_bytes(UART_NUM_0, p, (uint32_t)size, ticks);
        if (count < 0 || (size_t)count > size) return false;
        p += count; size -= (size_t)count;
    }
    return esp_timer_get_time() <= deadline_us;
}
bool ep_uart_write(const void *data, size_t size) {
    int count = uart_write_bytes(UART_NUM_0, data, size);
    return count >= 0 && (size_t)count == size &&
        uart_wait_tx_done(UART_NUM_0, pdMS_TO_TICKS(2000)) == ESP_OK;
}
esp_err_t ep_uart_discard(void) { return uart_flush_input(UART_NUM_0); }
