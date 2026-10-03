#!/usr/bin/env bash
set -euo pipefail
# Only this configuration is distributed. Full-only remains a development test.
source_dir=${FIRMWARE_BUILD_DIR:-firmware/esp32/build-partial}
output_dir=${RELEASE_OUTPUT_DIR:-build/release}
: "${RELEASE_COMMIT:?RELEASE_COMMIT must identify the checked source}"
[[ "$RELEASE_COMMIT" =~ ^[0-9a-f]{40}$ ]]
grep -qx 'CONFIG_EP_EXPERIMENTAL_PARTIAL=y' "$source_dir/sdkconfig"
test "$(stat -c %s "$source_dir/epaper_receiver.bin")" -le $((0x3e0000))
test -s "$source_dir/epaper_receiver.bin"
test ! -e "$output_dir/epaper-esp32-7in5-v2.tar.gz"
mkdir -p "$output_dir"
stage=$(mktemp -d)
trap 'rm -r -- "$stage"' EXIT
cp "$source_dir/epaper_receiver.bin" "$stage/firmware.bin"
cp docs/firmware-release.md "$stage/README.md"
cp LICENSE "$stage/LICENSE"
printf '%s\n' "$RELEASE_COMMIT" > "$stage/COMMIT"
# Application-only bundle: never package credential sectors, ELF, sdkconfig,
# a padded flash image, enrollment files, or a private build directory.
(cd "$stage" && sha256sum firmware.bin README.md LICENSE COMMIT > SHA256SUMS)
tar -C "$stage" -czf "$output_dir/epaper-esp32-7in5-v2.tar.gz" \
  firmware.bin README.md LICENSE COMMIT SHA256SUMS
cp "$stage/COMMIT" "$output_dir/COMMIT"
(cd "$output_dir" && sha256sum epaper-esp32-7in5-v2.tar.gz COMMIT > SHA256SUMS)
