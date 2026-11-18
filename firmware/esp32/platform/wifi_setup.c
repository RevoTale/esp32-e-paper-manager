#include "wifi_setup.h"
#include "radio.h"
#include "driver/gpio.h"
#include "esp_http_server.h"
#include "esp_system.h"
#include "esp_timer.h"
#include "esp_wifi.h"
#include "mbedtls/md.h"
#include "mbedtls/platform_util.h"
#include "freertos/FreeRTOS.h"
#include "freertos/queue.h"
#include "freertos/task.h"
#include <stdio.h>
#include <string.h>

typedef struct { uint8_t body[97]; size_t size; } submission;
static QueueHandle_t inbox;
static const char page[] =
    "<meta name=viewport content='width=device-width,initial-scale=1'><title>E-paper Wi-Fi</title>"
    "<form id=f><label>Wi-Fi name <input id=s required maxlength=32></label><p>"
    "<label>Password <input id=p type=password required minlength=8 maxlength=63></label><p>"
    "<button>Connect display</button></form><p id=r></p><script>f.onsubmit=async e=>{"
    "e.preventDefault();const enc=new TextEncoder(),a=enc.encode(s.value),b=enc.encode(p.value);"
    "if(a.length>32||b.length>63){r.textContent='Network name or password too long';return}"
    "try{const res=await fetch('/wifi',{method:'POST',headers:{'Content-Type':'application/octet-stream'},"
    "body:new Uint8Array([a.length,b.length,...a,...b])});r.textContent=await res.text()}"
    "catch(e){r.textContent='Connection closed. Check whether the display connected.'}}</script>";

static bool header(httpd_req_t *req, const char *name, const char *value) {
    char actual[64];
    return httpd_req_get_hdr_value_str(req, name, actual, sizeof actual) == ESP_OK && !strcmp(actual, value);
}
static esp_err_t index_page(httpd_req_t *req) {
    if (!header(req,"Host","192.168.4.1")) return httpd_resp_send_err(req,HTTPD_400_BAD_REQUEST,"Invalid host");
    httpd_resp_set_hdr(req,"Cache-Control","no-store");
    httpd_resp_set_hdr(req,"X-Frame-Options","DENY");
    return httpd_resp_send(req,page,HTTPD_RESP_USE_STRLEN);
}
static esp_err_t submit(httpd_req_t *req) {
    submission msg = {0};
    if (!header(req,"Host","192.168.4.1") || !header(req,"Origin","http://192.168.4.1") ||
        !header(req,"Content-Type","application/octet-stream") || req->content_len > sizeof msg.body ||
        req->content_len < 11) return httpd_resp_send_err(req,HTTPD_400_BAD_REQUEST,"Invalid request");
    while (msg.size < req->content_len) {
        int n = httpd_req_recv(req,(char *)msg.body + msg.size,req->content_len - msg.size);
        if (n <= 0) { mbedtls_platform_zeroize(&msg,sizeof msg); return ESP_FAIL; }
        msg.size += (size_t)n;
    }
    bool ok = msg.body[0] && msg.body[0] <= 32 && msg.body[1] >= 8 && msg.body[1] <= 63 &&
        msg.size == 2U + msg.body[0] + msg.body[1] && xQueueSend(inbox,&msg,0) == pdTRUE;
    mbedtls_platform_zeroize(&msg,sizeof msg);
    return ok ? httpd_resp_sendstr(req,"Trying the new network. The display will restart.") :
        httpd_resp_send_err(req,HTTPD_400_BAD_REQUEST,"Invalid network details");
}
static bool ap_config(const uint8_t *config, wifi_config_t *wifi) {
    uint8_t digest[32]; const char label[] = "epaper/mobile-ap/v1";
    if (mbedtls_md_hmac(mbedtls_md_info_from_type(MBEDTLS_MD_SHA256),config+40,32,
                        (const uint8_t *)label,sizeof label-1,digest)) return false;
    snprintf((char *)wifi->ap.ssid,sizeof wifi->ap.ssid,"Epaper-%02x%02x%02x",config[24],config[25],config[26]);
    for (unsigned i=0;i<16;i++) snprintf((char *)wifi->ap.password+2*i,3,"%02x",digest[i]);
    mbedtls_platform_zeroize(digest,sizeof digest);
    wifi->ap.authmode = WIFI_AUTH_WPA2_PSK; wifi->ap.pairwise_cipher = WIFI_CIPHER_TYPE_CCMP;
    wifi->ap.max_connection = 1; wifi->ap.channel = 1;
    return true;
}
static bool receive_network(const uint8_t *cfg, submission *msg) {
    wifi_config_t ap = {0}; httpd_handle_t server = NULL;
    httpd_config_t limits = HTTPD_DEFAULT_CONFIG();
    limits.recv_wait_timeout = 2; limits.send_wait_timeout = 2;
    const httpd_uri_t get = {.uri="/",.method=HTTP_GET,.handler=index_page};
    const httpd_uri_t post = {.uri="/wifi",.method=HTTP_POST,.handler=submit};
    inbox = xQueueCreate(1,sizeof *msg);
    bool ready = inbox && ap_config(cfg,&ap) && esp_netif_create_default_wifi_ap() &&
        esp_wifi_set_mode(WIFI_MODE_AP) == ESP_OK && esp_wifi_set_config(WIFI_IF_AP,&ap) == ESP_OK &&
        httpd_start(&server,&limits) == ESP_OK && httpd_register_uri_handler(server,&get) == ESP_OK &&
        httpd_register_uri_handler(server,&post) == ESP_OK;
    mbedtls_platform_zeroize(&ap,sizeof ap);
    bool received = ready && xQueueReceive(inbox,msg,pdMS_TO_TICKS(120000)) == pdTRUE;
    if (server) httpd_stop(server);
    if (inbox) { vQueueDelete(inbox); inbox=NULL; }
    return received;
}
static bool requested(void) {
    // Waveshare Rev3 S1/BOOT is IO0. Sample only after the ROM has booted.
    if (gpio_set_direction(GPIO_NUM_0, GPIO_MODE_INPUT) != ESP_OK) return false;
    unsigned held = 0;
    for (unsigned i=0;i<30;i++) {
        held = gpio_get_level(GPIO_NUM_0) == 0 ? held+1 : 0;
        if (held >= 6) return true;
        vTaskDelay(pdMS_TO_TICKS(100));
    }
    return false;
}
void ep_wifi_setup_boot(const ep_flash *flash, const uint8_t config[512], uint64_t generation) {
    if (memcmp(config+72,"tls://",6) || !requested()) return;
    submission msg = {0};
    if (ep_radio_open(config) && receive_network(config,&msg) && esp_wifi_set_mode(WIFI_MODE_STA) == ESP_OK) {
        uint8_t candidate[512]; memcpy(candidate,config,sizeof candidate);
        candidate[17] = msg.body[0]; candidate[18] = msg.body[1];
        memset(candidate+327,0,95); memcpy(candidate+327,msg.body+2,candidate[17]);
        memcpy(candidate+359,msg.body+2+candidate[17],candidate[18]);
        if (ep_config_valid(candidate) && ep_radio_configure(candidate) && ep_radio_join() && ep_radio_matches(candidate))
            (void)ep_config_save_wifi(flash,candidate,generation);
        mbedtls_platform_zeroize(candidate,sizeof candidate);
    }
    mbedtls_platform_zeroize(&msg,sizeof msg);
    esp_restart();
}
