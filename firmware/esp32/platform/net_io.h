#pragma once
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
int ep_net_dial(const char *host, const char *port, uint64_t connection);
bool ep_net_read(int fd, void *, size_t, int64_t deadline_us, uint64_t connection);
bool ep_net_write(int fd, const void *, size_t, int64_t deadline_us, uint64_t connection);
