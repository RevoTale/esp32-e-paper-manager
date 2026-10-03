#include "provision.h"
#include <string.h>

enum { PAGE = 512, SECTOR = 4096, CODE_INVALID_STATE = 2, CODE_STORAGE = 3 };
static bool valid_record(const uint8_t *p) {
    return !memcmp(p, "EPC2", 4) && p[4] == 1 && ep_be64(p + 8) != 0 &&
        ep_be32(p + 508) == ep_crc32(p, 508) && ep_config_valid(p);
}
int ep_config_load(const ep_flash *flash, uint8_t config[PAGE], uint64_t *generation) {
    uint8_t page[PAGE]; bool blank = true;
    memset(config, 0, PAGE); *generation = 0;
    if (!flash || !flash->read) return 3;
    for (size_t slot = 0; slot < 2; slot++) {
        if (flash->read(flash->context, slot * SECTOR, page, PAGE)) {
            memset(config, 0, PAGE); *generation = 0; return 3;
        }
        blank = blank && ep_filled(page, PAGE, 255);
        if (valid_record(page) && ep_be64(page + 8) > *generation) {
            memcpy(config, page, PAGE); *generation = ep_be64(page + 8);
        }
    }
    return *generation ? 1 : (blank ? 0 : 2);
}
bool ep_provision_valid(const uint8_t p[PAGE]) {
    if (memcmp(p, "EPCQ", 4) || (p[4] != 2 && p[4] != 3) || p[5] < 1 || p[5] > 5 ||
        ep_be32(p + 508) != ep_crc32(p, 508)) return false;
    // v3 correlation occupies only reserved wire bytes, never stored EPC2 data.
    if (p[4] == 3) {
        if (ep_filled(p + 486, 16, 0) || !ep_filled(p + 502, 6, 0)) return false;
    } else if (!ep_filled(p + 486, 22, 0)) return false;
    if (p[5] != 2 && p[5] != 3) return ep_filled(p + 6, 480, 0);
    if (!ep_config_valid(p)) return false;
    size_t lengths[] = {(size_t)p[20] * 256 + p[21], p[17], p[18], p[19]};
    const size_t starts[] = {72, 327, 359, 422}, capacity[] = {255, 32, 63, 64};
    if (!ep_filled(p + 6, 10, 0) || !ep_filled(p + 22, 2, 0)) return false;
    for (unsigned i = 0; i < 4; i++)
        if (!ep_filled(p + starts[i] + lengths[i], capacity[i] - lengths[i], 0)) return false;
    return true;
}
static int save(const ep_flash *flash, const uint8_t *request, uint64_t generation) {
    if (generation == UINT64_MAX) return CODE_STORAGE;
    uint8_t page[PAGE], verify[PAGE];
    memset(page, 255, PAGE); memcpy(page, "EPC2", 4); page[4] = 1;
    ep_put_be64(page + 8, generation + 1);
    memcpy(page + 16, request + 16, 470); memset(page + 486, 0, 22);
    ep_put_be32(page + 508, ep_crc32(page, 508));
    size_t offset = (size_t)((generation + 1) & 1) * SECTOR;
    if (flash->erase(flash->context, offset, SECTOR) ||
        flash->write(flash->context, offset, page, PAGE) ||
        flash->read(flash->context, offset, verify, PAGE)) return CODE_STORAGE;
    return memcmp(page, verify, PAGE) ? CODE_STORAGE : 0;
}
static void public_reply(uint8_t *out, const uint8_t *request, uint8_t code, int state,
                         const uint8_t *config, uint64_t generation) {
    memset(out, 0, PAGE); memcpy(out, "EPCR", 4);
    out[4] = request[4]; out[5] = request[5]; out[6] = (uint8_t)state; out[12] = code;
    if (request[4] == 3) memcpy(out + 392, request + 486, 16);
    if (state == 1) {
        out[7] = config[16]; out[8] = config[17]; out[9] = config[19];
        out[10] = config[20]; out[11] = config[21];
        ep_put_be64(out + 16, generation); memcpy(out + 24, config + 24, 16);
        memcpy(out + 40, config + 327, config[17]);
        memcpy(out + 72, config + 72, (size_t)config[20] * 256 + config[21]);
        memcpy(out + 327, config + 422, config[19]);
    }
    ep_put_be32(out + 508, ep_crc32(out, 508));
}
int ep_provision_readonly(const ep_flash *flash, const uint8_t request[PAGE], uint8_t reply[PAGE]) {
    if (!ep_provision_valid(request)) return -1;
    uint8_t config[PAGE]; uint64_t generation;
    int state = ep_config_load(flash, config, &generation);
    public_reply(reply, request, state == 3 ? CODE_STORAGE : 5, state, config, generation);
    return 0;
}
int ep_provision(const ep_flash *flash, const uint8_t request[PAGE], uint8_t reply[PAGE]) {
    if (!ep_provision_valid(request)) return -1;
    uint8_t config[PAGE]; uint64_t generation;
    int state = ep_config_load(flash, config, &generation);
    uint64_t expected_generation = generation + 1;
    int code = state == 3 ? CODE_STORAGE : 0;
    if (request[5] == 2 || request[5] == 3) {
        bool allowed = request[5] == 2 ? state == 0 || state == 2 :
            state == 1 && memcmp(config + 24, request + 24, 16) == 0;
        if (!code) code = allowed ? save(flash, request, generation) : CODE_INVALID_STATE;
    } else if (request[5] == 4) {
        if (code || !flash->erase || flash->erase(flash->context, 0, 2 * SECTOR)) code = CODE_STORAGE;
    }
    state = ep_config_load(flash, config, &generation);
    if (state == 3) code = CODE_STORAGE;
    // An acknowledged flash call is not proof of the authoritative state.
    // Match provision.Service.Report: verify the operation's postcondition.
    if (!code && request[5] == 4 && state != 0) code = CODE_STORAGE;
    if (!code && (request[5] == 2 || request[5] == 3) &&
        (state != 1 || generation != expected_generation ||
         memcmp(config + 16, request + 16, 470))) code = CODE_STORAGE;
    public_reply(reply, request, (uint8_t)code, state, config, generation);
    return 0;
}
