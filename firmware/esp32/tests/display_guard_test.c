#include "display_guard.h"
#include "esp_system.h"
#include <assert.h>
#include <stdio.h>

static esp_reset_reason_t reason;
static int begin_result, commit_result, abort_result;
static unsigned begins, commits, aborts;
esp_reset_reason_t esp_reset_reason(void) { return reason; }
static int begin(void *context) {
    (void)context;
    assert(ep_display_recovery_required()); // Guard must precede the first physical action.
    begins++; return begin_result;
}
static int write_pixels(void *context, uint8_t pass, uint32_t offset, const uint8_t *pixels, size_t size) {
    (void)context; (void)pass; (void)offset; (void)pixels; (void)size;
    assert(ep_display_recovery_required()); return 0;
}
static int commit(void *context) {
    (void)context; assert(ep_display_recovery_required()); commits++; return commit_result;
}
static int abort_frame(void *context) {
    (void)context; assert(ep_display_recovery_required()); aborts++; return abort_result;
}
static ep_sink fresh(void) {
    reason = ESP_RST_POWERON;
    assert(!ep_display_recovery_required());
    reason = ESP_RST_SW;
    begin_result = commit_result = abort_result = 0; begins = commits = aborts = 0;
    return ep_display_guard((ep_sink){NULL, begin, write_pixels, commit, abort_frame});
}
static void successful_lifecycle(void) {
    ep_sink sink = fresh(); uint8_t byte = 0;
    assert(!ep_display_recovery_required());
    assert(sink.begin(sink.context) == 0 && ep_display_recovery_required());
    assert(sink.write(sink.context, 0, 0, &byte, 1) == 0);
    assert(sink.commit(sink.context) == 0 && !ep_display_recovery_required());
    assert(begins == 1 && commits == 1 && aborts == 0);
    assert(sink.begin(sink.context) == 0);
    assert(sink.abort(sink.context) == 0 && !ep_display_recovery_required());
}
static void failure_retention(void) {
    const esp_reset_reason_t resets[] = {ESP_RST_SW, ESP_RST_PANIC, ESP_RST_INT_WDT,
        ESP_RST_TASK_WDT, ESP_RST_WDT, ESP_RST_DEEPSLEEP, ESP_RST_BROWNOUT, ESP_RST_UNKNOWN};
    for (unsigned i = 0; i < sizeof resets / sizeof resets[0]; i++) {
        ep_sink sink = fresh(); begin_result = -1;
        assert(sink.begin(sink.context) != 0);
        reason = resets[i];
        assert(ep_display_recovery_required());
        abort_result = -1;
        assert(sink.abort(sink.context) != 0 && ep_display_recovery_required());
        abort_result = 0;
        assert(sink.abort(sink.context) == 0 && !ep_display_recovery_required());
    }
    ep_sink sink = fresh(); assert(sink.begin(sink.context) == 0);
    commit_result = -1;
    assert(sink.commit(sink.context) != 0 && ep_display_recovery_required());
    assert(sink.abort(sink.context) == 0 && !ep_display_recovery_required());
    assert(commits == 1 && aborts == 1);
}
int main(void) {
    successful_lifecycle(); failure_retention();
    puts("display guard: set-before-begin, retain-on-error and clear-after-success passed (host logic only)");
}
