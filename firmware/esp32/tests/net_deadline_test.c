#include "net_io.h"
#include <assert.h>
#include <errno.h>
#include <stdio.h>
#include <sys/select.h>
#include <sys/socket.h>

typedef enum { FRAGMENT, INTERRUPT, RETRY, CANCEL, OVERRUN, EOF_PEER } scenario;
static scenario current;
static int64_t now;
static unsigned waits, transfers;
static bool live;

int64_t esp_timer_get_time(void) { return now; }
bool ep_network_live(uint64_t connection) { return live && connection == 1; }

int __wrap_select(int n, fd_set *readers, fd_set *writers, fd_set *errors, struct timeval *timeout) {
    (void)errors;
    assert(n == 4 && ((readers != NULL) != (writers != NULL)));
    assert(timeout != NULL && timeout->tv_sec == 0 && timeout->tv_usec <= 25000);
    assert(++waits <= 3); // An accidentally renewed deadline cannot spin forever.
    now += current == OVERRUN ? 150000 : 20000;
    if (current == CANCEL) live = false;
    if (current == INTERRUPT) { errno = EINTR; return -1; }
    return 1;
}

ssize_t __wrap_recv(int fd, void *bytes, size_t size, int flags) {
    assert(fd == 3 && size > 0 && flags == 0);
    transfers++;
    if (current == RETRY) { errno = EAGAIN; return -1; }
    if (current == EOF_PEER) return 0;
    *(unsigned char *)bytes = 0x5a;
    return 1;
}

ssize_t __wrap_send(int fd, const void *bytes, size_t size, int flags) {
    assert(bytes != NULL);
    unsigned char ignored;
    return __wrap_recv(fd, &ignored, size, flags);
}

static void reset(scenario value) {
    current = value; now = 1000; waits = 0; transfers = 0; live = true;
}

static void fixed_deadline(bool writing, scenario value) {
    reset(value);
    unsigned char bytes[8] = {0};
    int64_t deadline = now + 50000;
    bool result = writing ? ep_net_write(3, bytes, sizeof bytes, deadline, 1)
                          : ep_net_read(3, bytes, sizeof bytes, deadline, 1);
    assert(!result);
    assert(waits == 3 && now == 61000);
    assert(transfers == (value == INTERRUPT ? 0 : 2));
    if (!writing && value == FRAGMENT) {
        assert(bytes[0] == 0x5a && bytes[1] == 0x5a && bytes[2] == 0);
    }
}

static void reject_late_or_cancelled(bool writing, scenario value) {
    reset(value);
    unsigned char byte = 0;
    bool result = writing ? ep_net_write(3, &byte, 1, now + 50000, 1)
                          : ep_net_read(3, &byte, 1, now + 50000, 1);
    assert(!result && waits == 1 && transfers == 0 && byte == 0);
}

static void success_and_expiry(void) {
    reset(FRAGMENT);
    unsigned char bytes[2] = {0};
    assert(ep_net_read(3, bytes, sizeof bytes, now + 50000, 1));
    assert(waits == 2 && transfers == 2 && bytes[0] == 0x5a && bytes[1] == 0x5a);
    reset(FRAGMENT);
    assert(!ep_net_read(3, bytes, 1, now, 1));
    assert(!ep_net_write(3, bytes, 1, now - 1, 1));
    assert(waits == 0 && transfers == 0);
    reset(EOF_PEER);
    assert(!ep_net_read(3, bytes, 1, now + 50000, 1));
    assert(waits == 1 && transfers == 1);
}

int main(void) {
    for (unsigned writing = 0; writing < 2; writing++) {
        fixed_deadline(writing != 0, FRAGMENT);
        fixed_deadline(writing != 0, INTERRUPT);
        fixed_deadline(writing != 0, RETRY);
        reject_late_or_cancelled(writing != 0, CANCEL);
        reject_late_or_cancelled(writing != 0, OVERRUN);
    }
    success_and_expiry();
    puts("fixed deadlines: fragments, EINTR, EAGAIN, cancellation and late readiness passed");
}
