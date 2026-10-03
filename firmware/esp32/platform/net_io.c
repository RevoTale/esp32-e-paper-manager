#include "net_io.h"
#include "network_state.h"
#include "esp_timer.h"
#include "lwip/sockets.h"
#include "lwip/netdb.h"
#include <errno.h>
#include <fcntl.h>
#include <unistd.h>

static bool ready(int fd, bool writing, int64_t deadline, uint64_t connection) {
    while (ep_network_live(connection) && esp_timer_get_time() < deadline) {
        fd_set set; FD_ZERO(&set); FD_SET(fd, &set);
        struct timeval timeout = {.tv_sec = 0, .tv_usec = 25000};
        int result = select(fd + 1, writing ? NULL : &set, writing ? &set : NULL, NULL, &timeout);
        if (result > 0) return ep_network_live(connection) && esp_timer_get_time() < deadline;
        if (result < 0 && errno != EINTR) return false;
    }
    return false;
}
static bool transfer(int fd, void *bytes, size_t size, int64_t deadline, uint64_t connection, bool writing) {
    uint8_t *cursor = bytes;
    while (size) {
        if (!ready(fd, writing, deadline, connection)) return false;
        ssize_t count = writing ? send(fd, cursor, size, 0) : recv(fd, cursor, size, 0);
        if (count < 0 && (errno == EAGAIN || errno == EWOULDBLOCK || errno == EINTR)) continue;
        if (count <= 0 || (size_t)count > size) return false;
        cursor += count; size -= (size_t)count;
    }
    return ep_network_live(connection) && esp_timer_get_time() <= deadline;
}
bool ep_net_read(int fd, void *bytes, size_t n, int64_t deadline, uint64_t connection) {
    return transfer(fd, bytes, n, deadline, connection, false);
}
bool ep_net_write(int fd, const void *bytes, size_t n, int64_t deadline, uint64_t connection) {
    return transfer(fd, (void *)bytes, n, deadline, connection, true);
}
int ep_net_dial(const char *host, const char *port, uint64_t connection) {
    struct addrinfo hints = {.ai_family = AF_INET, .ai_socktype = SOCK_STREAM}, *addresses;
    if (getaddrinfo(host, port, &hints, &addresses) != 0) return -1;
    int socket_fd = -1;
    int64_t deadline = esp_timer_get_time() + 10000000;
    for (struct addrinfo *a = addresses; a && ep_network_live(connection); a = a->ai_next) {
        int fd = socket(a->ai_family, a->ai_socktype, a->ai_protocol);
        if (fd < 0) continue;
        int flags = fcntl(fd, F_GETFL, 0);
        bool ok = flags >= 0 && fcntl(fd, F_SETFL, flags | O_NONBLOCK) == 0;
        if (ok && connect(fd, a->ai_addr, a->ai_addrlen) < 0) {
            ok = errno == EINPROGRESS && ready(fd, true, deadline, connection);
            int error = 0; socklen_t size = sizeof error;
            ok = ok && getsockopt(fd, SOL_SOCKET, SO_ERROR, &error, &size) == 0 && !error;
        }
        if (ok) { socket_fd = fd; break; }
        close(fd);
    }
    freeaddrinfo(addresses); return socket_fd;
}
