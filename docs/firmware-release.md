# ESP32 7.5-inch V2 application update

One release contains one full/partial-capable ESP32 application and the manager
container with the same version tag. The internal `build-partial` name is a
development detail, not a second firmware product. This targets the verified
ESP32 e-Paper Driver Board Rev3 and monochrome 800x480 panel only.

This archive is an **application update**, not first-install media. Existing
bootloader and partition layout must match `firmware/esp32/partitions.csv` in
the release source. A new/unprovisioned board needs the installation procedure
in `firmware/esp32/README.md`; do not use these instructions for another layout.

1. Verify the downloaded archive against the release `SHA256SUMS`.
2. Extract it; verify its internal `SHA256SUMS` before flashing.
3. Stop the manager/serial clients. Use the project's approved flasher to write
   **only `firmware.bin` at `0x10000`**. Verify the write using that flasher.
4. Restart the matching manager and confirm capabilities, then visible output.

Never use erase-all or a padded whole-flash image. Provisioning/epoch state at
`0x3fc000..0x3fffff` and NVS are not included. Partial remains an explicit
manager opt-in (`-refresh-policy -experimental-partial`); full-only operation
uses the same application. Keep the consecutive-partial and periodic-full rules.

The user accepted the working prototype on October1. The interrupted soak test
is not continuous endurance acceptance; see `docs/partial-candidate-20261001.md`.
Release automation does not establish panel lifetime, power or fault-recovery
qualification. Releases remain marked prerelease pending those boundaries.

Source and dependency license notices accompany the tagged repository/SDK;
the bundled project MIT license does not replace upstream component licenses.
