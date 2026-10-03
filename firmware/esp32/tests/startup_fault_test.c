#include "panel75.h"
#include "provision.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

static void missing_storage(void) {
    const ep_flash missing = {0};
    const ep_flash *backends[] = {NULL, &missing};
    for (unsigned backend = 0; backend < 2; backend++) {
        uint8_t config[512]; uint64_t generation = 99;
        memset(config, 0xa6, sizeof config);
        assert(ep_config_load(backends[backend], config, &generation) == 3);
        assert(generation == 0 && ep_filled(config, sizeof config, 0));
        for (uint8_t operation = 1; operation <= 5; operation++) {
            uint8_t request[512] = {0}, reply[512];
            memcpy(request, "EPCQ", 4); request[4] = 2; request[5] = operation;
            if (operation == 2 || operation == 3) {
                request[16] = 2; request[17] = 1; request[18] = 8; request[19] = 3; request[21] = 9;
                memset(request + 24, 1, 48); memcpy(request + 72, "tcp://a:1", 9);
                request[327] = 'a'; memcpy(request + 359, "password", 8); memcpy(request + 422, "UTC", 3);
            }
            ep_put_be32(request + 508, ep_crc32(request, 508));
            assert(ep_provision_valid(request));
            assert(ep_provision_readonly(backends[backend], request, reply) == 0);
            assert(reply[6] == 3 && reply[12] == 3 && ep_filled(reply + 16, 492, 0));
            assert(ep_provision(backends[backend], request, reply) == 0);
            assert(reply[6] == 3 && reply[12] == 3 && ep_filled(reply + 16, 492, 0));
            assert(ep_be32(reply + 508) == ep_crc32(reply, 508));
        }
    }
}
static uint8_t screen_send(ep_screen *screen, ep_record request, uint64_t now) {
    uint8_t bytes[120]; ep_record reply;
    size_t size = ep_screen_handle(screen, &request, now, bytes, sizeof bytes);
    assert(ep_wire_decode(bytes, size, &reply));
    return reply.payload[1];
}
static void missing_panel_io(void) {
    ep_panel75 panel; ep_screen screen;
    ep_sink sink = ep_panel75_create(&panel, (ep_panel_io){0});
    uint8_t boot[16] = {1}, id[16] = {2}, claim[32] = {1}, digest[32] = {0}, pixel = 0;
    memset(claim + 16, 3, 16);
    assert(ep_screen_init(&screen, (ep_screen_config){800, 480, 1000, 1, 2, 1, 180000}, sink, boot, id));
    screen.fatal = true;
    assert(screen_send(&screen, (ep_record){.kind=1}, 0) == 0);
    assert(screen_send(&screen, (ep_record){.kind=2, .size=32, .payload=claim}, 0) == 0);
    assert(screen_send(&screen, (ep_record){.kind=3, .epoch=1, .size=32, .payload=claim}, 0) == 0);
    assert(screen_send(&screen, (ep_record){.kind=4, .epoch=1, .id=1, .size=32, .payload=digest}, 180000) == 11);
    assert(screen_send(&screen, (ep_record){.kind=5, .epoch=1, .id=1, .size=1, .payload=&pixel}, 180001) == 7);
    assert(screen_send(&screen, (ep_record){.kind=8, .epoch=1}, 180002) == 11);
    ep_screen_disconnect(&screen);
    assert(ep_screen_tick(&screen, 999999) == 0);
    uint8_t trace[36], diagnostic[11];
    ep_panel_status(&panel, trace); ep_panel_diagnostic(&panel, diagnostic);
    assert(trace[0] == 1 && ep_filled(trace + 1, 35, 0) && ep_filled(diagnostic, 11, 0));
    assert(!panel.active && panel.cycle == 0); // No invalid callback was ever entered.
}
static uint64_t now;
static bool busy_low;
static int spi(void *context, const uint8_t *bytes, size_t size) { (void)context; (void)bytes; (void)size; return 0; }
static int pin(void *context, ep_panel_pin name, bool high) { (void)context; (void)name; (void)high; return 0; }
static int busy(void *context) { (void)context; return busy_low ? 0 : 1; }
static uint64_t clock_us(void *context) { (void)context; return now; }
static void delay_us(void *context, uint32_t delay) { (void)context; now += delay; }
static void first_failure_diagnostics(void) {
    ep_panel75 panel;
    ep_sink sink = ep_panel75_create(&panel, (ep_panel_io){NULL, spi, pin, busy, clock_us, delay_us});
    busy_low = true; now = 0;
    assert(sink.begin(sink.context) != 0 && panel.failed && panel.command == 4);
    uint8_t before[11], after[11]; ep_panel_diagnostic(&panel, before);
    uint32_t power_samples = panel.samples[0];
    assert(power_samples && before[0] == 1 && before[1] == 2 && before[4] == 4);
    busy_low = false;
    assert(sink.abort(sink.context) == 0 && !panel.active);
    ep_panel_diagnostic(&panel, after);
    assert(!memcmp(before, after, sizeof before));
    assert(panel.samples[0] == power_samples); // Shutdown cannot become a power-on sample.
}
int main(void) {
    missing_storage(); missing_panel_io(); first_failure_diagnostics();
    puts("startup faults: null storage, unavailable panel and first-failure diagnostics passed");
}
