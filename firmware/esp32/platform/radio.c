#include "radio.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_wifi.h"
#include "esp_netif.h"
#include "esp_event.h"
#include "esp_timer.h"
#include "mbedtls/platform_util.h"
#include <stdatomic.h>
#include <string.h>

static atomic_bool online;
static bool secure_ap(void) {
    wifi_ap_record_t ap;
    if (esp_wifi_sta_get_ap_info(&ap) != ESP_OK) return false;
    return (ap.authmode == WIFI_AUTH_WPA2_PSK || ap.authmode == WIFI_AUTH_WPA2_WPA3_PSK) &&
        ap.pairwise_cipher == WIFI_CIPHER_TYPE_CCMP && ap.group_cipher == WIFI_CIPHER_TYPE_CCMP;
}
static void event(void *context, esp_event_base_t base, int32_t id, void *data) {
    (void)context; (void)data;
    if (base == WIFI_EVENT && id == WIFI_EVENT_STA_DISCONNECTED) atomic_store(&online, false);
    if (base == IP_EVENT && id == IP_EVENT_STA_LOST_IP) atomic_store(&online, false);
    if ((base == IP_EVENT && id == IP_EVENT_STA_GOT_IP) ||
        (base == WIFI_EVENT && id == WIFI_EVENT_STA_CONNECTED) ||
        (base == WIFI_EVENT && id == WIFI_EVENT_STA_AUTHMODE_CHANGE)) {
        bool safe = secure_ap();
        if (!safe) { atomic_store(&online, false); esp_wifi_disconnect(); }
        else if (base == IP_EVENT) atomic_store(&online, true);
    }
}
bool ep_radio_open(const uint8_t config[512]) {
    // ESP-IDF v5.5.5 Wi-Fi station APIs; credentials remain in RAM, no NVS erase.
    // https://docs.espressif.com/projects/esp-idf/en/v5.5.5/esp32/api-reference/network/esp_wifi.html
    if (esp_netif_init() != ESP_OK || esp_event_loop_create_default() != ESP_OK ||
        !esp_netif_create_default_wifi_sta()) return false;
    wifi_init_config_t init = WIFI_INIT_CONFIG_DEFAULT(); init.nvs_enable = 0;
    if (esp_wifi_init(&init) != ESP_OK) return false;
    if (esp_event_handler_register(WIFI_EVENT, ESP_EVENT_ANY_ID, event, NULL) != ESP_OK ||
        esp_event_handler_register(IP_EVENT, ESP_EVENT_ANY_ID, event, NULL) != ESP_OK) return false;
    wifi_config_t station = {0};
    memcpy(station.sta.ssid, config + 327, config[17]);
    memcpy(station.sta.password, config + 359, config[18]);
    station.sta.scan_method = WIFI_ALL_CHANNEL_SCAN;
    station.sta.sort_method = WIFI_CONNECT_AP_BY_SECURITY;
    station.sta.threshold.authmode = WIFI_AUTH_WPA2_PSK;
    station.sta.threshold.rssi = -127;
    station.sta.pmf_cfg.capable = true;
    // WPA2 compatibility, never WPA1/TKIP; negotiated AP admission above is
    // stricter than authmode's numeric threshold (which includes WPA/WPA2 mixed).
    esp_err_t result = esp_wifi_set_storage(WIFI_STORAGE_RAM);
    if (!result) result = esp_wifi_set_mode(WIFI_MODE_STA);
    if (!result) result = esp_wifi_set_config(WIFI_IF_STA, &station);
    mbedtls_platform_zeroize(&station, sizeof station);
    if (!result) result = esp_wifi_set_ps(WIFI_PS_MIN_MODEM);
    if (!result) result = esp_wifi_start();
    return result == ESP_OK;
}
bool ep_radio_online(void) { return atomic_load(&online); }
bool ep_radio_join(void) {
    if (ep_radio_online()) return true;
    if (esp_wifi_connect() != ESP_OK) return false;
    int64_t deadline = esp_timer_get_time() + 30000000;
    while (!ep_radio_online() && esp_timer_get_time() < deadline) vTaskDelay(pdMS_TO_TICKS(100));
    if (ep_radio_online()) return true;
    esp_wifi_disconnect(); return false;
}
