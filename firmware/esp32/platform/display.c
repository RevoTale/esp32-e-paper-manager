#include "display.h"
#include "driver/gpio.h"
#include "driver/spi_master.h"
#include "esp_rom_sys.h"
#include "esp_timer.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"

// Rev3 schematic: GPIO4 power gating is NOT verified; never drive it.
// https://files.waveshare.com/wiki/E-Paper-ESP32-Driver-Board/E-Paper_ESP32_Driver_Board_V3.pdf
static spi_device_handle_t device;
static int transfer(void *context, const uint8_t *bytes, size_t size) {
    (void)context;
    if (!size || size > 64) return -1;
    spi_transaction_t transaction = {.length = size * 8, .tx_buffer = bytes};
    return spi_device_transmit(device, &transaction);
}
static int pin(void *context, ep_panel_pin pin, bool high) {
    (void)context;
    const gpio_num_t pins[] = {GPIO_NUM_15, GPIO_NUM_27, GPIO_NUM_26};
    return (unsigned)pin < 3 ? gpio_set_level(pins[pin], high ? 1 : 0) : -1;
}
static int busy(void *context) { (void)context; return gpio_get_level(GPIO_NUM_25); }
static uint64_t clock_us(void *context) { (void)context; return (uint64_t)esp_timer_get_time(); }
static void delay_us(void *context, uint32_t us) {
    (void)context;
    // Keep the documented 2 ms reset LOW pulse, even with a 10 ms RTOS tick.
    // Longer waits yield the core; round upward, never shorten panel settling.
    if (us < 10000) { esp_rom_delay_us(us); return; }
    uint32_t tick_us = portTICK_PERIOD_MS * 1000U;
    vTaskDelay((us + tick_us - 1) / tick_us);
}
esp_err_t ep_display_open(ep_panel_io *io) {
    gpio_config_t outputs = {.pin_bit_mask = (1ULL << 15) | (1ULL << 27) | (1ULL << 26),
        .mode = GPIO_MODE_OUTPUT, .intr_type = GPIO_INTR_DISABLE};
    // Preload output latches before enabling the pins; CS and RESET start HIGH.
    esp_err_t result = gpio_set_level(GPIO_NUM_15, 1);
    if (!result) result = gpio_set_level(GPIO_NUM_26, 1);
    if (!result) result = gpio_set_level(GPIO_NUM_27, 0);
    if (!result) result = gpio_config(&outputs);
    gpio_config_t input = {.pin_bit_mask = 1ULL << 25, .mode = GPIO_MODE_INPUT,
        .intr_type = GPIO_INTR_DISABLE};
    if (!result) result = gpio_config(&input);
    if (result) return result;
    spi_bus_config_t bus = {.mosi_io_num = 14, .miso_io_num = -1, .sclk_io_num = 13,
        .quadwp_io_num = -1, .quadhd_io_num = -1, .max_transfer_sz = 64};
    // Interrupt-backed transmit yields instead of polling every pixel transfer.
    result = spi_bus_initialize(SPI2_HOST, &bus, SPI_DMA_DISABLED);
    if (result) return result;
    const spi_device_interface_config_t config = {.clock_speed_hz = 1000000, .mode = 0,
        .spics_io_num = -1, .queue_size = 1, .flags = SPI_DEVICE_HALFDUPLEX};
    result = spi_bus_add_device(SPI2_HOST, &config, &device);
    if (result) { spi_bus_free(SPI2_HOST); return result; }
    *io = (ep_panel_io){NULL, transfer, pin, busy, clock_us, delay_us};
    return ESP_OK;
}
