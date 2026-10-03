#include "provision.h"
#include <string.h>

static bool text(const uint8_t *p, size_t size, size_t min, size_t max) {
    if (size < min || size > max) return false;
    for (size_t i = 0; i < size;) {
        uint32_t cp = p[i++]; unsigned continuation = 0; uint32_t minimum = 0;
        if (cp >= 0xc2 && cp <= 0xdf) { continuation = 1; cp &= 31; minimum = 0x80; }
        else if (cp >= 0xe0 && cp <= 0xef) { continuation = 2; cp &= 15; minimum = 0x800; }
        else if (cp >= 0xf0 && cp <= 0xf4) { continuation = 3; cp &= 7; minimum = 0x10000; }
        else if (cp >= 0x80) return false;
        if (continuation > size - i) return false;
        for (unsigned j = 0; j < continuation; j++) {
            uint8_t c = p[i++]; if ((c & 0xc0) != 0x80) return false;
            cp = (cp << 6) | (c & 63);
        }
        if (cp < minimum || cp > 0x10ffff || (cp >= 0xd800 && cp <= 0xdfff) || cp < 32 || cp == 127)
            return false;
    }
    return true;
}
static bool alnum(uint8_t c) {
    return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9');
}
static bool manager(const uint8_t *p, size_t size) {
    if (size < 9 || size > 255 || memcmp(p, "tcp://", 6)) return false;
    size_t i = 6, label = 0;
    for (; i < size && p[i] != ':'; i++) {
        if (p[i] == '.') {
            if (!label || p[i-1] == '-') return false;
            label = 0;
        } else {
            if ((!alnum(p[i]) && p[i] != '-') || (!label && p[i] == '-') || ++label > 63) return false;
        }
    }
    if (!label || i == size || p[i-1] == '-' || i-6 > 253 || ++i == size) return false;
    uint32_t port = 0;
    for (; i < size; i++) {
        if (p[i] < '0' || p[i] > '9') return false;
        port = port * 10 + (uint32_t)(p[i] - '0');
        if (port > 65535) return false;
    }
    return port != 0;
}
static bool timezone(const uint8_t *p, size_t size) {
    if (!size || size > 64 || p[0] == '/' || p[size-1] == '/') return false;
    for (size_t i = 0; i < size; i++) {
        if (!alnum(p[i]) && p[i] != '_' && p[i] != '+' && p[i] != '-' && p[i] != '.' && p[i] != '/') return false;
        if (i && ((p[i] == '.' && p[i-1] == '.') || (p[i] == '/' && p[i-1] == '/'))) return false;
    }
    return true;
}
bool ep_config_valid(const uint8_t p[512]) {
    size_t manager_size = (size_t)p[20] * 256 + p[21];
    return p[16] == 2 && text(p + 327, p[17], 1, 32) && text(p + 359, p[18], 8, 63) &&
        timezone(p + 422, p[19]) && manager(p + 72, manager_size) &&
        !ep_filled(p + 24, 16, 0) && !ep_filled(p + 40, 32, 0);
}
