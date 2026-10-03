#include "uart.h"
#include "driver/uart.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

static int64_t now;
static int read_result;
static int64_t resume_time;
static unsigned read_calls;
static TickType_t last_ticks;

int64_t esp_timer_get_time(void) { return now; }
int uart_read_bytes(int port, void *out, uint32_t size, TickType_t ticks) {
    assert(port == UART_NUM_0 && ticks > 0);
    ++read_calls;
    last_ticks = ticks;
    if (read_result > 0 && (uint32_t)read_result <= size)
        memset(out, 'E', (size_t)read_result);
    now = resume_time;
    return read_result;
}
esp_err_t uart_param_config(int port, const uart_config_t *config) {
    (void)port; (void)config; return ESP_OK;
}
esp_err_t uart_set_pin(int port, int tx, int rx, int rts, int cts) {
    (void)port; (void)tx; (void)rx; (void)rts; (void)cts; return ESP_OK;
}
esp_err_t uart_driver_install(int port, int rx, int tx, int queue_size,
                              void *queue, int flags) {
    (void)port; (void)rx; (void)tx; (void)queue_size; (void)queue;
    (void)flags; return ESP_OK;
}
int uart_write_bytes(int port, const void *data, size_t size) {
    (void)port; (void)data; (void)size; return -1;
}
esp_err_t uart_wait_tx_done(int port, TickType_t ticks) {
    (void)port; (void)ticks; return ESP_OK;
}
esp_err_t uart_flush_input(int port) { (void)port; return ESP_OK; }

static void prepare(int result, int64_t resumed_at) {
    now = 0; read_result = result; resume_time = resumed_at; read_calls = 0;
}
static void incomplete_and_failed_reads(void) {
    uint8_t out[2] = {0};
    prepare(1, 20001);
    assert(!ep_uart_read(out, sizeof(out), 20000));
    assert(out[0] == 'E' && out[1] == 0 && read_calls == 1);
    prepare(0, 20001);
    assert(!ep_uart_read(out, sizeof(out), 20000));
    assert(read_calls == 1);
    prepare(-1, 1);
    assert(!ep_uart_read(out, sizeof(out), 20000));
    prepare(3, 1);
    assert(!ep_uart_read(out, sizeof(out), 20000));
    prepare(1, 1);
    now = 20000;
    assert(!ep_uart_read(out, sizeof(out), 20000));
    assert(read_calls == 0);
}
static void completed_records_enforce_deadline(void) {
    uint8_t out[2] = {0};
    prepare(1, 19999);
    assert(ep_uart_read(out, 1, 20000));
    prepare(1, 20000);
    assert(ep_uart_read(out, 1, 20000));
    prepare(2, 20001);
    assert(!ep_uart_read(out, sizeof(out), 20000));
    assert(out[0] == 'E' && out[1] == 'E');
}
static void polling_preserves_consumed_bytes(void) {
    uint8_t out = 0;
    // UART has consumed the first protocol byte before the task resumes.
    // Returning false would silently discard it in USB first-byte polling.
    prepare(1, 20001);
    assert(ep_uart_poll_byte(&out));
    assert(out == 'E' && read_calls == 1 && last_ticks == pdMS_TO_TICKS(20));
    prepare(0, 20001);
    assert(!ep_uart_poll_byte(&out));
    prepare(-1, 1);
    assert(!ep_uart_poll_byte(&out));
    prepare(2, 1);
    assert(!ep_uart_poll_byte(&out));
}
int main(void) {
    incomplete_and_failed_reads();
    completed_records_enforce_deadline();
    polling_preserves_consumed_bytes();
    puts("uart_read: PASS");
    return 0;
}
