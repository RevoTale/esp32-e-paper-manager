#include "net_io.h"
#include "wire.h"
#include <assert.h>
#include <errno.h>
#include <fcntl.h>
#include <netinet/in.h>
#include <pthread.h>
#include <signal.h>
#include <stdatomic.h>
#include <stdio.h>
#include <string.h>
#include <sys/socket.h>
#include <time.h>
#include <unistd.h>

static atomic_uint_fast64_t live_connection = 1;
// Deadline correctness is mandatory in net_deadline_test; scheduler latency is
// qualified separately with the same socket scenarios and original ceilings.
static void qualify_latency(const char *operation, int64_t elapsed, int64_t ceiling) {
#ifdef EP_QUALIFY_LATENCY
    fprintf(stderr, "%s: elapsed=%lld us, ceiling=%lld us\n",
            operation, (long long)elapsed, (long long)ceiling);
    assert(elapsed < ceiling);
#else
    (void)operation; (void)elapsed; (void)ceiling;
#endif
}
int64_t esp_timer_get_time(void) {
    struct timespec now;
    assert(clock_gettime(CLOCK_MONOTONIC, &now) == 0);
    return (int64_t)now.tv_sec * 1000000 + now.tv_nsec / 1000;
}
bool ep_network_live(uint64_t connection) { return connection == atomic_load(&live_connection); }
static void pause_us(long microseconds) {
    struct timespec delay = {.tv_sec = microseconds / 1000000, .tv_nsec = microseconds % 1000000 * 1000};
    while (nanosleep(&delay, &delay)) assert(errno == EINTR);
}
static void pair(int sockets[2]) {
    assert(socketpair(AF_UNIX, SOCK_STREAM, 0, sockets) == 0);
    for (unsigned i = 0; i < 2; i++) {
        int flags = fcntl(sockets[i], F_GETFL, 0);
        assert(flags >= 0 && fcntl(sockets[i], F_SETFL, flags | O_NONBLOCK) == 0);
    }
    atomic_store(&live_connection, 1);
}
static void close_pair(int sockets[2]) { assert(close(sockets[0]) == 0); assert(close(sockets[1]) == 0); }
typedef struct { int fd; size_t size; bool reading; long pause; } worker_args;
static void *fragment_worker(void *context) {
    worker_args *work = context; uint8_t bytes[733]; memset(bytes, 0x5a, sizeof bytes);
    size_t completed = 0; int64_t deadline = esp_timer_get_time() + 3000000;
    while (completed < work->size) {
        assert(esp_timer_get_time() < deadline);
        size_t size = work->reading ? sizeof bytes : 1;
        if (size > work->size - completed) size = work->size - completed;
        ssize_t count = work->reading ? recv(work->fd, bytes, size, 0) : send(work->fd, bytes, size, 0);
        if (count < 0) { assert(errno == EAGAIN || errno == EWOULDBLOCK || errno == EINTR); pause_us(1000); continue; }
        assert(count > 0);
        if (work->reading) assert(ep_filled(bytes, (size_t)count, 0x5a));
        completed += (size_t)count;
        pause_us(work->pause);
    }
    return NULL;
}
static void eof_and_expiry(void) {
    int sockets[2]; pair(sockets); uint8_t output[8]; memset(output, 0xa5, sizeof output);
    assert(shutdown(sockets[1], SHUT_WR) == 0);
    assert(!ep_net_read(sockets[0], output, sizeof output, esp_timer_get_time() + 50000, 1));
    assert(ep_filled(output, sizeof output, 0xa5)); close_pair(sockets);
    pair(sockets); assert(send(sockets[1], "abc", 3, 0) == 3); assert(shutdown(sockets[1], SHUT_WR) == 0);
    assert(!ep_net_read(sockets[0], output, sizeof output, esp_timer_get_time() + 50000, 1));
    assert(!memcmp(output, "abc", 3) && ep_filled(output + 3, 5, 0xa5)); close_pair(sockets);
    pair(sockets); assert(send(sockets[1], "z", 1, 0) == 1);
    assert(!ep_net_read(sockets[0], output, 1, esp_timer_get_time() - 1, 1));
    assert(ep_net_read(sockets[0], output, 1, esp_timer_get_time() + 50000, 1) && output[0] == 'z');
    assert(!ep_net_write(sockets[0], output, 1, esp_timer_get_time() - 1, 1));
    assert(ep_net_read(sockets[0], output, 0, esp_timer_get_time() + 50000, 1));
    close_pair(sockets);
}
static void fragmented_reads_and_writes(void) {
    int sockets[2]; pthread_t thread; uint8_t bytes[65536]; pair(sockets);
    worker_args work = {sockets[1], 32, false, 500};
    assert(pthread_create(&thread, NULL, fragment_worker, &work) == 0);
    assert(ep_net_read(sockets[0], bytes, 32, esp_timer_get_time() + 1000000, 1));
    assert(ep_filled(bytes, 32, 0x5a)); assert(pthread_join(thread, NULL) == 0); close_pair(sockets);
    pair(sockets); int buffer_size = 4096;
    assert(setsockopt(sockets[0], SOL_SOCKET, SO_SNDBUF, &buffer_size, sizeof buffer_size) == 0);
    memset(bytes, 0x5a, sizeof bytes); work = (worker_args){sockets[1], sizeof bytes, true, 1000};
    assert(pthread_create(&thread, NULL, fragment_worker, &work) == 0);
    assert(ep_net_write(sockets[0], bytes, sizeof bytes, esp_timer_get_time() + 2000000, 1));
    assert(pthread_join(thread, NULL) == 0); close_pair(sockets);
}
static void shared_deadline(void) {
    int sockets[2]; pair(sockets); pthread_t thread; uint8_t bytes[8];
    worker_args work = {sockets[1], sizeof bytes, false, 20000};
    assert(pthread_create(&thread, NULL, fragment_worker, &work) == 0);
    int64_t began = esp_timer_get_time();
    assert(!ep_net_read(sockets[0], bytes, sizeof bytes, began + 50000, 1));
    qualify_latency("shared read deadline", esp_timer_get_time() - began, 120000);
    assert(pthread_join(thread, NULL) == 0); close_pair(sockets);
    pair(sockets); uint8_t block[4096] = {0};
    while (send(sockets[0], block, sizeof block, 0) > 0) {}
    assert(errno == EAGAIN || errno == EWOULDBLOCK);
    began = esp_timer_get_time();
    assert(!ep_net_write(sockets[0], block, sizeof block, began + 50000, 1));
    qualify_latency("shared write deadline", esp_timer_get_time() - began, 120000);
    close_pair(sockets);
}
static void *cancel_worker(void *context) {
    (void)context; pause_us(20000); atomic_store(&live_connection, 2); return NULL;
}
static void cancellation(void) {
    int sockets[2]; pair(sockets); pthread_t thread; uint8_t byte;
    assert(pthread_create(&thread, NULL, cancel_worker, NULL) == 0);
    int64_t began = esp_timer_get_time();
    assert(!ep_net_read(sockets[0], &byte, 1, began + 1000000, 1));
    qualify_latency("cancellation", esp_timer_get_time() - began, 150000);
    assert(pthread_join(thread, NULL) == 0);
    assert(!ep_net_write(sockets[0], "x", 1, esp_timer_get_time() + 50000, 1));
    assert(!ep_net_read(sockets[0], &byte, 0, esp_timer_get_time() + 50000, 1));
    close_pair(sockets);
}
static void local_dial(void) {
    int listener = socket(AF_INET, SOCK_STREAM, 0); assert(listener >= 0);
    struct sockaddr_in address = {.sin_family = AF_INET, .sin_addr.s_addr = htonl(INADDR_LOOPBACK)};
    assert(bind(listener, (struct sockaddr *)&address, sizeof address) == 0 && listen(listener, 1) == 0);
    socklen_t size = sizeof address; assert(getsockname(listener, (struct sockaddr *)&address, &size) == 0);
    char port[6]; assert(snprintf(port, sizeof port, "%u", ntohs(address.sin_port)) > 0);
    atomic_store(&live_connection, 1);
    int client = ep_net_dial("127.0.0.1", port, 1); assert(client >= 0);
    assert(fcntl(client, F_GETFL, 0) & O_NONBLOCK);
    int accepted = accept(listener, NULL, NULL); assert(accepted >= 0);
    assert(close(accepted) == 0 && close(client) == 0 && close(listener) == 0);
    assert(ep_net_dial("127.0.0.1", port, 1) == -1);
    atomic_store(&live_connection, 2); assert(ep_net_dial("127.0.0.1", port, 1) == -1);
}
static void caller_length_boundary(void) {
    int sockets[2]; pair(sockets); uint8_t header[EP_HEADER] = {0}, incoming[EP_HEADER];
    memcpy(header, "EPS2", 4); header[4] = 5;
    ep_put_le(header + 8, 1, 8); ep_put_le(header + 16, 1, 8);
    // The transport may read only the fixed header before validating its length.
    header[6] = 1;
    assert(ep_wire_size(header) == EP_HEADER + 1);
    header[6] = 0xff; header[7] = 0xff;
    assert(send(sockets[1], header, sizeof header, 0) == sizeof header);
    assert(ep_net_read(sockets[0], incoming, sizeof incoming, esp_timer_get_time() + 50000, 1));
    assert(ep_wire_size(incoming) == 0); // No peer-sized allocation/read occurs.
    close_pair(sockets);
}
int main(void) {
    // POSIX send raises SIGPIPE; lwIP reports the socket error without this signal.
    assert(signal(SIGPIPE, SIG_IGN) != SIG_ERR);
    eof_and_expiry(); fragmented_reads_and_writes(); shared_deadline(); cancellation(); local_dial(); caller_length_boundary();
    puts("net I/O faults: EOF, partial transfer, shared deadline, cancellation and bounded framing passed");
}
