#include "runtime.h"
#include "network.h"
#include "esp_timer.h"
#include "esp_task_wdt.h"
#include "mbedtls/platform_util.h"
#include <string.h>

static uint64_t now_ms(void) { return (uint64_t)esp_timer_get_time() / 1000; }
static void hardware_failure(ep_runtime *r, ep_message *reply) {
    ep_record record;
    if (!ep_wire_decode(reply->bytes, reply->size, &record)) { reply->size = 0; return; }
    uint8_t body[88]; memcpy(body, record.payload, record.size);
    ep_panel_diagnostic(&r->panel, body + 9);
    if (!body[9]) { // Initialization/recovery refusal, not a fabricated BUSY/SPI sample.
        body[9] = 1; body[10] = 4; ep_put_le(body + 16, UINT32_MAX, 4);
    }
    record.payload = body;
    reply->size = (uint16_t)ep_wire_encode(reply->bytes, sizeof reply->bytes, &record);
}
static void diagnostic(ep_runtime *r, const ep_record *request, ep_message *reply) {
    ep_record record;
    if (!ep_wire_decode(reply->bytes, reply->size, &record)) { reply->size = 0; return; }
    uint8_t body[84]; memcpy(body, record.payload, 48); body[1] = 0;
    if (request->kind == 10) { ep_network_health(body + 48); record.size = 56; }
    else { ep_panel_status(&r->panel, body + 48); record.size = 84; }
    record.payload = body;
    reply->size = (uint16_t)ep_wire_encode(reply->bytes, sizeof reply->bytes, &record);
}
static void dispatch(ep_runtime *r, ep_source source, const ep_message *request, ep_message *reply) {
    reply->connection = request->connection; reply->size = 0;
    if (source == EP_NETWORK && !ep_network_live(request->connection)) return;
    if (!ep_owner_admit(&r->owner, source, request->connection, now_ms())) return;
    if (source == EP_USB) ep_network_block(true);
    if (source == EP_USB && request->size == 512 && !memcmp(request->bytes, "EPCQ", 4)) {
        ep_screen_disconnect(&r->screen);
        // A failed write may still change authoritative storage. Never resume
        // a network session using the old key after ANY mutation attempt.
        if (request->bytes[5] >= 2 && request->bytes[5] <= 4) r->network_fenced = true;
        int result = r->boot_safe ? ep_provision(&r->flash, request->bytes, reply->bytes) :
            ep_provision_readonly(&r->flash, request->bytes, reply->bytes);
        if (!result) reply->size = 512;
    } else {
        ep_record record;
        if (!ep_wire_decode(request->bytes, request->size, &record) || record.kind == 9) return;
        uint64_t started = now_ms();
        reply->size = (uint16_t)ep_screen_handle(&r->screen, &record, started, reply->bytes, sizeof reply->bytes);
        if (reply->size) {
            uint8_t code = reply->bytes[EP_HEADER + 1];
            ep_screen_completed(&r->screen, record.kind, code, started, now_ms());
            reply->size = (uint16_t)ep_screen_reply(&r->screen, &record, code, reply->bytes, sizeof reply->bytes);
            if (code == 11) hardware_failure(r, reply);
            if (record.kind == 10 || record.kind == 12) diagnostic(r, &record, reply);
        }
    }
    ep_owner_completed(&r->owner, now_ms());
}
void ep_runtime_run(ep_runtime *r) {
    ep_message request, reply;
    for (;;) {
        esp_task_wdt_reset();
        ep_owner_tick(&r->owner, now_ms());
        ep_network_block(r->network_fenced || r->owner.source == EP_USB);
        ep_channel *channel = &r->usb; ep_source source = EP_USB;
        if (xQueueReceive(channel->requests, &request, 0) != pdTRUE) {
            channel = &r->network; source = EP_NETWORK;
            if (xQueueReceive(channel->requests, &request, pdMS_TO_TICKS(10)) != pdTRUE) continue;
        }
        if (!request.size) ep_owner_closed(&r->owner, source, request.connection);
        else {
            dispatch(r, source, &request, &reply);
            // Never let a dead transport block the owner; connection-tagged ACK.
            xQueueOverwrite(channel->replies, &reply);
        }
        mbedtls_platform_zeroize(&request, sizeof request);
        mbedtls_platform_zeroize(&reply, sizeof reply);
    }
}
