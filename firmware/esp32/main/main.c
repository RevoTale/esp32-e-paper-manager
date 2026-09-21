#include "runtime.h"
#include "storage.h"
#include "uart.h"
#include "usb_receiver.h"
#include "display.h"
#include "display_guard.h"
#include "network.h"
#include "esp_mac.h"
#include "esp_task_wdt.h"
#include "mbedtls/platform_util.h"
#include "freertos/task.h"

static ep_runtime runtime;
static void stop(void) { for (;;) vTaskDelay(portMAX_DELAY); }
void app_main(void) {
    ep_runtime *r = &runtime;
    if (ep_uart_open() != ESP_OK) { stop(); return; }
    uint64_t epoch = 0, generation;
    r->boot_safe = ep_storage_open(&r->flash) == ESP_OK &&
        ep_epoch_reserve(&r->flash, &epoch) == EP_STORE_OK;
    uint8_t config[512], boot[16] = {'E', '3'}, id[16] = {0};
    int stored = ep_config_load(&r->flash, config, &generation);
    if (esp_read_mac(boot + 2, ESP_MAC_WIFI_STA) != ESP_OK) r->boot_safe = false;
    ep_put_be64(boot + 8, epoch);
    if (stored == 1) for (unsigned i = 0; i < 16; i++) id[i] = config[24 + i];
    ep_panel_io io = {0};
    bool recovery = ep_display_recovery_required();
    bool display_ready = ep_display_open(&io) == ESP_OK;
    ep_sink sink = ep_display_guard(ep_panel75_create(&r->panel, io));
    const ep_screen_config profile = {800, 480, 1000, 1, 2, 1, 180000};
    if (!ep_screen_init(&r->screen, profile, sink, boot, id)) { stop(); return; }
    // Keep USB Inspect/Health alive on storage/display initialization failure.
    // A fatal screen rejects Begin before any sink callback can touch hardware.
    r->screen.fatal = recovery || !r->boot_safe || !display_ready;
    r->owner.screen = &r->screen;
    if (!ep_channel_open(&r->usb) || !ep_channel_open(&r->network)) { stop(); return; }
    if (!ep_usb_start(&r->usb)) { stop(); return; }
    if (r->boot_safe && stored == 1) ep_network_start(&r->network, config, epoch);
    mbedtls_platform_zeroize(config, sizeof config);
    const esp_task_wdt_config_t watchdog = {.timeout_ms = 60000, .idle_core_mask = 0, .trigger_panic = true};
    if (esp_task_wdt_reconfigure(&watchdog) != ESP_OK || esp_task_wdt_add(NULL) != ESP_OK) { stop(); return; }
    ep_runtime_run(r);
}
