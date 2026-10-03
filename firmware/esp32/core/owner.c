#include "owner.h"
void ep_owner_closed(ep_owner *o, ep_source source, uint64_t connection) {
    if (source != EP_NONE && o->source == source && o->connection == connection) {
        ep_screen_disconnect(o->screen); o->source = EP_NONE; o->connection = 0;
    }
}
void ep_owner_tick(ep_owner *o, uint64_t now) {
    if (now < o->observed) return;
    o->observed = now;
    ep_screen_tick(o->screen, now);
    if (o->source == EP_USB && now >= o->last_usb && now - o->last_usb >= 20000)
        ep_owner_closed(o, EP_USB, o->connection);
}
void ep_owner_completed(ep_owner *o, uint64_t now) {
    if (now < o->observed) return;
    o->observed = now;
    if (o->source == EP_USB) o->last_usb = now;
}
bool ep_owner_admit(ep_owner *o, ep_source source, uint64_t connection, uint64_t now) {
    if ((source != EP_USB && source != EP_NETWORK) || !connection || now < o->observed) return false;
    o->observed = now;
    if (o->source == EP_USB && source == EP_NETWORK) return false;
    if (o->source != source || o->connection != connection) {
        if (connection <= o->highest[source]) return false;
        ep_owner_closed(o, o->source, o->connection);
        o->source = source; o->connection = connection;
        o->highest[source] = connection;
    }
    if (source == EP_USB) o->last_usb = now;
    return true;
}
