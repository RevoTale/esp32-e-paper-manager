#include "screen.h"
void ep_screen_completed(ep_screen *s, uint8_t operation, uint8_t code,
                         uint64_t entered, uint64_t completed) {
    if (completed < entered || entered < s->observed) {
        if (s->state == 1 || s->state == 2) ep_screen_fail(s, 10);
        return;
    }
    s->observed = completed;
    if (operation == 6 && s->refresh_pending) {
        s->last_refresh = completed; s->refresh_pending = false;
    }
    if (code) return;
    if ((operation == 4 || operation == 5 || operation == 13 || operation == 14) &&
        (s->state == 1 || s->state == 2)) {
        uint64_t elapsed = completed - entered;
        if (elapsed > UINT64_MAX - s->started) { ep_screen_fail(s, 10); return; }
        // Only bounded synchronous sink execution is excluded. Requests arriving
        // slowly still consume the full idle/total budget; queries cannot renew it.
        s->started += elapsed; s->progress = completed;
    }
}
