#include "owner.h"
#include <assert.h>
static unsigned closes, ticks;
void ep_screen_disconnect(ep_screen *s) { (void)s; closes++; }
uint8_t ep_screen_tick(ep_screen *s, uint64_t now) { (void)s; (void)now; ticks++; return 0; }
int main(void) {
    ep_screen screen; ep_owner owner = {.screen = &screen};
    assert(!ep_owner_admit(&owner, EP_NONE, 1, 0));
    assert(!ep_owner_admit(&owner, EP_USB, 0, 0));
    assert(ep_owner_admit(&owner, EP_NETWORK, 1, 1) && !closes);
    assert(ep_owner_admit(&owner, EP_NETWORK, 1, 2) && !closes);
    assert(ep_owner_admit(&owner, EP_USB, 1, 3) && closes == 1);
    assert(!ep_owner_admit(&owner, EP_NETWORK, 2, 4));
    ep_owner_closed(&owner, EP_NETWORK, 1); assert(closes == 1);
    assert(ep_owner_admit(&owner, EP_USB, 2, 5) && closes == 2);
    ep_owner_closed(&owner, EP_USB, 1); assert(closes == 2);
    assert(!ep_owner_admit(&owner, EP_USB, 2, 4)); // time regression
    ep_owner_tick(&owner, 20004); assert(owner.source == EP_USB);
    ep_owner_tick(&owner, 20005); assert(owner.source == EP_NONE && closes == 3);
    assert(ep_owner_admit(&owner, EP_NETWORK, 2, 20006));
    ep_owner_closed(&owner, EP_NETWORK, 2); assert(closes == 4 && owner.source == EP_NONE);
    assert(!ep_owner_admit(&owner, EP_NETWORK, 2, 20007));
    assert(!ep_owner_admit(&owner, EP_NETWORK, 1, 20008));
    assert(!ep_owner_admit(&owner, EP_NETWORK, 3, 2));
    assert(ep_owner_admit(&owner, EP_USB, 3, 20009));
    ep_owner_completed(&owner, 50009);
    ep_owner_tick(&owner, 50010); assert(owner.source == EP_USB);
    assert(ticks == 3);
}
