#include "storage.h"
#include "esp_flash.h"
#include "esp_partition.h"

enum { CHIP_BYTES = 0x400000, STORE_START = 0x3fc000, STORE_BYTES = 0x4000 };
static bool valid_range(size_t offset, size_t size, size_t alignment) {
    return size && offset < STORE_BYTES && size <= STORE_BYTES - offset &&
           offset % alignment == 0 && size % alignment == 0;
}
static int read_store(void *context, size_t offset, void *out, size_t size) {
    if (!out || !valid_range(offset, size, 1)) return ESP_ERR_INVALID_ARG;
    return esp_partition_read(context, offset, out, size);
}
static int write_store(void *context, size_t offset, const void *in, size_t size) {
    if (!in || !valid_range(offset, size, 32)) return ESP_ERR_INVALID_ARG;
    return esp_partition_write(context, offset, in, size);
}
static int erase_store(void *context, size_t offset, size_t size) {
    if (!valid_range(offset, size, 4096)) return ESP_ERR_INVALID_ARG;
    return esp_partition_erase_range(context, offset, size);
}
esp_err_t ep_storage_open(ep_flash *flash) {
    if (!flash) return ESP_ERR_INVALID_ARG;
    *flash = (ep_flash){0};
    uint32_t size = 0;
    esp_err_t result = esp_flash_get_physical_size(NULL, &size);
    if (result != ESP_OK) return result;
    const esp_partition_t *partition = esp_partition_find_first(
        ESP_PARTITION_TYPE_DATA, ESP_PARTITION_SUBTYPE_ANY, "epaper");
    if (size != CHIP_BYTES || !partition || partition->address != STORE_START ||
        partition->size != STORE_BYTES || partition->erase_size != 4096 ||
        partition->encrypted || partition->readonly) return ESP_ERR_INVALID_STATE;
    // ESP-IDF owns cache/interrupt/core coordination; no direct ROM writes.
    // https://docs.espressif.com/projects/esp-idf/en/v5.5.5/esp32/api-reference/storage/partition.html
    *flash = (ep_flash){(void *)partition, read_store, write_store, erase_store};
    return ESP_OK;
}
