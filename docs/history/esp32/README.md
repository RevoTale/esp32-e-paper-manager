# ESP32 Driver Board Rev3 panel diagnostics

Status: 7.5-inch V2 diagnostic visibly accepted; 1.54-inch B V2 not accepted.
Neither diagnostic is the complete remote-display firmware.
Pico and Pi5 remain unchanged. Current investigation: [port notes](../docs/esp32-port.md).

## Accepted 7.5-inch checkpoint

On 2026-09-12 the operator confirmed the black rectangle on white after one
`t` using `panel75-check.bin`, SHA256
`70b0d5f9b6b4b333c6af604e0c0df18fc9372ec1cc147d05570bf1dbd5b17908`.
Direct macOS flashing verified the image, then UART returned
`EP75-V2 CHECK v1 READY`. The final controller-result log was not recovered;
visible confirmation is independent evidence, not a reconstructed USB result.

Operator-confirmed switches: display selector **A/ON**, USB-to-UART **ON**.
The latter enables the serial bridge; it is not a master power switch.
Keep this tested assembly unchanged. Conflicting revision-specific A/B tables
must not be generalized to another board; see the debugging history.
This accepts one frame, not repeat updates, streaming, Wi-Fi or measured energy.
Full receiver work follows [the plan](../remote-epaper/tasks/esp32-75.md).

## Build

Inside the existing TinyGo 0.42.0 Dev Container:

```sh
cd /workspaces/pico-sandbox/experiments/03-remote-epaper
tinygo build -target=esp32-coreboard-v2 -size=short -o ../12-esp32-epaper/panel-check.bin ./cmd/esp32-panel-check
```

For the previously accepted 7.5-inch **monochrome V2 800x480** panel, use the
separate profile (never use this image on the 1.54-inch tricolor panel):

```sh
tinygo build -target=esp32-coreboard-v2 -tags=epaper75 -size=short -o ../12-esp32-epaper/panel75-check.bin ./cmd/esp32-panel-check
```

This reuses `panel.Driver` without changing its controller sequence or Pico
adapter. The 7.5 profile sends one 48000-byte frame through both controller
planes, waits for BUSY HIGH, and issues controller power-off/deep-sleep.
It draws a black rectangle at x40..399/y40..439 on white. It accepts only one
`t` attempt per boot. ESP32 Rev3 supply control is not assumed: power callback
logs explicitly describe the manual supply, not successful electrical cutoff.
Disconnect power on failure. The SPI adapter and command diagnostics are shared;
the original `panel-check.bin` artifact remains preserved.

Original ESP32-WROOM-32E uses this target; do not select ESP32-S3 or Pico.
Image builds alone do not prove panel compatibility or wiring.

## First hardware acceptance

Disconnect power before inserting or moving the FPC. Confirm the board's
display-config resistor is 3R for 1.54-inch (B), not the old 7.5-inch setting.
Rev3 uses different A/B meanings from the separate Raspberry Pi HAT; never
transfer the old switch instruction. Board photo/label confirmation is pending.

Connect the board's USB data connector. It is a CH343P serial bridge, not a
UF2 disk. With the exact serial device identified, TinyGo can build and flash:

```sh
tinygo flash -target=esp32-coreboard-v2 -port=/dev/ttyUSB0 ./cmd/esp32-panel-check
tinygo monitor -target=esp32-coreboard-v2 -port=/dev/ttyUSB0
```

`/dev/ttyUSB0` is a placeholder: inspect the actual assigned device first.
Close other serial clients before flashing. TinyGo's native ESP32 flasher is
selected by the target; no Python/esptool installation is needed for this path.
**Correction after source inspection:** TinyGo 0.42.0 `tinygo flash` calls
`EraseFlash` for this target. The commands above therefore erase the chip;
do not use them when preserving existing flash data. Prefer the standalone
flasher below, without `-erase-all`.
If automatic bootloader entry fails, use the board's documented FLASH/reset
procedure, not Pico BOOTSEL. Do not erase flash as a troubleshooting shortcut.

Send the single character `t`. Expect a white border with a black rectangle on
the left and red on the right. Serial prints an error with stage or
CONTROLLER_COMPLETE. There is no automatic refresh loop, no Wi-Fi and no token
in this diagnostic. Starts are separated by at least 180 seconds. Any hardware
failure latches the diagnostic: disconnect power rather than retry. The board
has no confirmed software power-cut path and sleep cannot be assumed on error.
This is a conservative diagnostic guard, not final scheduling policy.

Small-panel full acceptance still needs normal UART operation, the exact visible colors,
BUSY completion and a second successful frame after sleep. USB passthrough and
flashing were untested at this historical checkpoint; direct flashing is now
verified below. Dedicated passthrough remains unnecessary for this workflow.

## Remaining application work

Durable storage for identity/credentials/epochs, encrypted manager connection,
UART transport/provisioning, tricolor manager output, status timestamp and final
resource/security/physical tests remain. Do not deploy this diagnostic as the
network receiver or treat a successful Wi-Fi example build as authenticated
network acceptance.

## Verified direct macOS flashing

Build upstream `tinygo.org/x/espflasher@v0.8.1` for darwin/arm64 inside the
existing container; keep the executable local to this experiment. No global
installation or Python is necessary. With host serial access explicitly
authorized, run in this directory on macOS:

```sh
./espflasher-darwin-arm64 -port /dev/cu.usbmodem5B140746091 -chip esp32 -offset 0x1000 -baud 115200 panel75-check.bin
```

The port is the observed device, not a permanent identifier. Recheck on each
connection. This writes the selected image region, not an entire-chip erase.
The first 1.54 diagnostic was flashed this way; ESP32 and 4MB flash detected,
MD5 verified, reset completed. Its UART returned CONTROLLER_COMPLETE, but the
user saw no rectangles. This does not prove a broken panel.

Sources: [TinyGo flasher call](https://github.com/tinygo-org/tinygo/blob/v0.42.0/main.go),
[standalone flasher](https://github.com/tinygo-org/espflasher/tree/v0.8.1).
