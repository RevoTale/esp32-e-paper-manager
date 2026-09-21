#include "channel.h"
#include "esp_timer.h"
bool ep_channel_open(ep_channel *c) {
    c->requests = xQueueCreate(1, sizeof(ep_message));
    c->replies = xQueueCreate(1, sizeof(ep_message));
    if (c->requests && c->replies) return true;
    if (c->requests) vQueueDelete(c->requests);
    if (c->replies) vQueueDelete(c->replies);
    return false;
}
bool ep_channel_exchange(ep_channel *c, const ep_message *request, ep_message *reply) {
    // One in-flight request per transport. Stale replies cannot cross connections.
    if (xQueueSend(c->requests, request, pdMS_TO_TICKS(1000)) != pdTRUE) return false;
    int64_t deadline = esp_timer_get_time() + 65000000;
    while (esp_timer_get_time() < deadline) {
        if (xQueueReceive(c->replies, reply, pdMS_TO_TICKS(50)) == pdTRUE &&
            reply->connection == request->connection) return reply->size != 0;
    }
    return false;
}
void ep_channel_close(ep_channel *c, uint64_t connection) {
    const ep_message event = {.connection = connection};
    // Queue capacity is exactly one. Cancel a queued, not-yet-dispatched record
    // rather than blocking or losing its terminal disconnect behind that record.
    xQueueOverwrite(c->requests, &event);
}
