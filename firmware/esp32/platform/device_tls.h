#pragma once
#include "net_io.h"
bool ep_tls_start(int fd, const char *host, const char *hex, uint64_t connection);
void ep_tls_close(void);
bool ep_tls_transfer(int fd, void *buf, size_t len, int64_t deadline, uint64_t connection, bool writing);
