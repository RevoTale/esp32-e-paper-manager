#include "owner.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

static unsigned closes;
void ep_screen_disconnect(ep_screen *screen) { (void)screen; closes++; }
uint8_t ep_screen_tick(ep_screen *screen, uint64_t now) {
    if (now < screen->observed) return 10;
    screen->observed = now;
    return 0;
}
static void stale_replacement(void) {
    ep_screen screen = {0}; ep_owner owner = {.screen = &screen};
    assert(ep_owner_admit(&owner, EP_NETWORK, 2, 10));
    assert(!ep_owner_admit(&owner, EP_NETWORK, 1, 11));
    assert(owner.source == EP_NETWORK && owner.connection == 2 && closes == 0);
    assert(ep_owner_admit(&owner, EP_NETWORK, 2, 12));
    ep_owner_closed(&owner, EP_NETWORK, 1);
    assert(owner.connection == 2 && closes == 0);
    ep_owner_closed(&owner, EP_NETWORK, 2);
    assert(!ep_owner_admit(&owner, EP_NETWORK, 2, 13));
    assert(ep_owner_admit(&owner, EP_NETWORK, 3, 14));
}
static void preemption_resurrection(void) {
    ep_screen screen = {0}; ep_owner owner = {.screen = &screen};
    assert(ep_owner_admit(&owner, EP_NETWORK, 1, 1));
    assert(ep_owner_admit(&owner, EP_USB, 1, 2));
    assert(closes == 1);
    ep_owner_tick(&owner, 20002);
    assert(owner.source == EP_NONE && closes == 2);
    assert(!ep_owner_admit(&owner, EP_NETWORK, 1, 20003));
    assert(!ep_owner_admit(&owner, EP_USB, 1, 20003));
    assert(owner.source == EP_NONE && closes == 2);
    assert(ep_owner_admit(&owner, EP_NETWORK, 2, 20004));
    ep_owner_closed(&owner, EP_NETWORK, 1);
    assert(owner.connection == 2 && closes == 2);
}
static void regression_before_preemption(void) {
    ep_screen screen = {0}; ep_owner owner = {.screen = &screen};
    assert(ep_owner_admit(&owner, EP_NETWORK, 1, 100));
    assert(!ep_owner_admit(&owner, EP_USB, 1, 99));
    assert(owner.source == EP_NETWORK && owner.connection == 1 && closes == 0);
    assert(ep_owner_admit(&owner, EP_USB, 1, 101));
    ep_owner_tick(&owner, 20100);
    assert(owner.source == EP_USB);
    ep_owner_tick(&owner, 20101);
    assert(owner.source == EP_NONE);
}
static void maximum_connection(void) {
    ep_screen screen = {0}; ep_owner owner = {.screen = &screen};
    assert(ep_owner_admit(&owner, EP_NETWORK, UINT64_MAX, 1));
    ep_owner_closed(&owner, EP_NETWORK, UINT64_MAX);
    assert(!ep_owner_admit(&owner, EP_NETWORK, UINT64_MAX, 2));
    assert(!ep_owner_admit(&owner, EP_NETWORK, 0, 2));
    assert(!ep_owner_admit(&owner, EP_NETWORK, 1, 2));
}
static void completion_idle_floor(void) {
    ep_screen screen = {0}; ep_owner owner = {.screen = &screen};
    assert(ep_owner_admit(&owner, EP_USB, 1, 1000));
    ep_owner_completed(&owner, 31000);
    assert(owner.last_usb == 31000 && owner.observed == 31000);
    ep_owner_completed(&owner, 30000);
    assert(owner.last_usb == 31000 && owner.observed == 31000);
    ep_owner_tick(&owner, 50999);
    assert(owner.source == EP_USB && closes == 0);
    ep_owner_tick(&owner, 51000);
    assert(owner.source == EP_NONE && closes == 1);
}
int main(int argc, char **argv) {
    if (argc == 1 || !strcmp(argv[1], "stale")) stale_replacement();
    closes = 0;
    if (argc == 1 || !strcmp(argv[1], "resurrection")) preemption_resurrection();
    closes = 0;
    if (argc == 1 || !strcmp(argv[1], "clock")) regression_before_preemption();
    closes = 0;
    if (argc == 1 || !strcmp(argv[1], "maximum")) maximum_connection();
    closes = 0;
    if (argc == 1 || !strcmp(argv[1], "completion")) completion_idle_floor();
    puts("owner faults: stale generations, preemption and clock ordering passed");
}
