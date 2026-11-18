#include "device_tls.h"
#include <stdlib.h>
#include <sys/random.h>
#include <time.h>
#include <unistd.h>
#include <string.h>

bool ep_network_live(uint64_t connection) { return connection == 1; }
int64_t esp_timer_get_time(void) {
    struct timespec ts; clock_gettime(CLOCK_MONOTONIC, &ts);
    return (int64_t)ts.tv_sec*1000000 + ts.tv_nsec/1000;
}
void esp_fill_random(void *buf, size_t len) {
    if (getrandom(buf,len,0) != (ssize_t)len) abort();
}
int main(int argc, char **argv) {
    if (argc != 5) return 2;
    int fd=ep_net_dial(argv[1],argv[2],1);
    if (fd < 0) return 3;
    char msg[]="frame"; int64_t deadline=esp_timer_get_time()+5000000;
    bool ok=ep_tls_start(fd,argv[3],argv[4],1) &&
        ep_tls_transfer(fd,msg,5,deadline,1,true) && ep_tls_transfer(fd,msg,3,deadline,1,false) &&
        !memcmp(msg,"ack",3);
    ep_tls_close(); close(fd); return ok ? 0 : 1;
}
