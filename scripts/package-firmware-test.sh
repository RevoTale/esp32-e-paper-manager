#!/usr/bin/env bash
set -euo pipefail
fixture=$(mktemp -d)
trap 'rm -r -- "$fixture"' EXIT
mkdir "$fixture/source"
printf 'not-a-real-firmware\n' > "$fixture/source/epaper_receiver.bin"
printf 'CONFIG_EP_EXPERIMENTAL_PARTIAL=y\n' > "$fixture/source/sdkconfig"
printf 'must-not-leak\n' > "$fixture/source/enrollment.json"
export FIRMWARE_BUILD_DIR="$fixture/source" RELEASE_OUTPUT_DIR="$fixture/out"
export RELEASE_COMMIT=0123456789012345678901234567890123456789
bash scripts/package-firmware.sh
(cd "$fixture/out" && sha256sum --check SHA256SUMS)
mkdir "$fixture/unpacked"
tar -xzf "$fixture/out/epaper-esp32-7in5-v2.tar.gz" -C "$fixture/unpacked"
(cd "$fixture/unpacked" && sha256sum --check SHA256SUMS)
test "$(find "$fixture/unpacked" -type f | wc -l)" -eq 5
test ! -e "$fixture/unpacked/enrollment.json"
cmp "$fixture/source/epaper_receiver.bin" "$fixture/unpacked/firmware.bin"
if bash scripts/package-firmware.sh; then exit 1; fi # no asset overwrite
export RELEASE_OUTPUT_DIR="$fixture/rejected"
printf 'CONFIG_EP_EXPERIMENTAL_PARTIAL=n\n' > "$fixture/source/sdkconfig"
if bash scripts/package-firmware.sh; then exit 1; fi # wrong capability
printf 'CONFIG_EP_EXPERIMENTAL_PARTIAL=y\n' > "$fixture/source/sdkconfig"
truncate -s $((0x3e0001)) "$fixture/source/epaper_receiver.bin"
if bash scripts/package-firmware.sh; then exit 1; fi # credential overlap
