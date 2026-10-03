idf_component_register(SRCS "main.c" "runtime.c" "../core/bytes.c" "../core/epoch.c"
    "../core/config.c" "../core/provision.c" "../core/wire.c"
    "../core/secure.c" "../core/secure_record.c" "../core/screen.c"
    "../core/screen_frame.c" "../core/screen_reply.c" "../platform/storage.c" "../platform/uart.c"
    "../core/screen_region.c" "../core/region.c"
    "../core/panel75_region.c" "../core/panel75_window.c"
    "../core/panel75.c" "../core/panel75_io.c" "../platform/display.c"
    "../core/panel75_trace.c" "../core/owner.c" "../core/screen_timing.c"
    "../platform/channel.c" "../platform/usb_receiver.c" "../platform/display_guard.c"
    "../platform/network.c" "../platform/net_io.c" "../platform/radio.c"
    INCLUDE_DIRS "../core" "../platform"
    REQUIRES esp_partition spi_flash esp_driver_uart esp_timer mbedtls esp_driver_spi esp_driver_gpio
        esp_wifi esp_event esp_netif lwip esp_system esp_hw_support)
target_compile_options(${COMPONENT_LIB} PRIVATE -Wall -Wextra -Werror -Wconversion)

# UART carries authenticated binary records. Panic dumps also risk exposing
# credentials/register contents; stale local sdkconfig must not override this.
if(NOT CONFIG_ESP_SYSTEM_PANIC_SILENT_REBOOT OR NOT CONFIG_ESP_COREDUMP_ENABLE_TO_NONE)
    message(FATAL_ERROR "Receiver requires silent panic reboot and disabled core dumps")
endif()
