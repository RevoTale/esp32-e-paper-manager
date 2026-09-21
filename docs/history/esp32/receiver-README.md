# ESP32 receiver — 7.5-inch V2

Status: integrated firmware; **first USB and encrypted Wi-Fi frames visually
confirmed on hardware on 2026-09-20**. Interruption recovery and long-running
acceptance remain pending; this is not full release acceptance.
The known-good TinyGo diagnostic and Pico/Pi5 paths remain unchanged.

## Target and architecture

- Waveshare e-Paper ESP32 Driver Board Rev3, original ESP32-WROOM-32E,
  physical4MiB flash; the previously identified 7.5 V2 monochrome800×480 panel.
- Accepted switch checkpoint: A / ON. Do not apply this profile to the
  1.54-inch tri-colour or 2.13-inch V4 panel.
- ESP-IDF5.5.5 C receiver, under the user's C/WPA2 exception. Go renderer,
  manager, provisioning and USB tools remain Go. This is not a TinyGo binary.
- HTML/inline CSS, fonts, images, composition, clipping, timestamp and damage
  remain server responsibilities. ESP32 receives bounded EPS2 pixel records,
  never HTML, CSS or a browser runtime. Current device profile advertises full
  raw frames only; it does not promise partial refresh or compressed payloads.

| Component | Responsibility |
| --- | --- |
| `core/provision*`, `config` | Canonical EPCQ/EPCR, validated EPC2 credentials, authoritative read-back |
| `core/epoch` | Durable monotonic EPE1 boot epochs, fail closed on damage |
| `core/secure*`, `wire` | Existing Go EPN2 authentication/AES-GCM and bounded EPS2 records |
| `core/screen*` | Lease, transaction, digest, ordering, commit/reconciliation, cooldown |
| `core/owner*` | One hardware owner; USB priority, retired connection rejection |
| `core/panel75*` | Exact panel initialization, planes, BUSY waits, refresh, sleep, diagnostics |
| `platform` | ESP-IDF flash, UART, SPI/GPIO, Wi-Fi, socket I/O and task queues |
| `main/runtime` | Serializes all storage/panel mutations and routes replies |

The negotiated profile is800×480, stride100, two48000-byte passes, maximum
chunk1000, legacy full-refresh interval180seconds (including startup).
The new software candidate advertises configurable refresh policy; upgrade
the Go manager and USB tools with it. `-refresh-policy` enables a separately
bounded urgent lane, not partial refresh or BUSY bypass. Default normal180s /
urgent30s are operator policies; shorter intervals are not manufacturer safety
guarantees. See [the contract](../../03-remote-epaper/docs/refresh-priority.md).
The same logical1-bit bitmap is sent twice: old-plane bytes are inverted for
command0x10; new-plane bytes are unchanged for0x13. A64-byte scratch buffer
matches original ESP32 non-DMA SPI limits; no additional48000-byte framebuffer
is allocated. This trades additional wire bytes for bounded MCU RAM, not a
measured energy improvement.

## Safety and ownership

- SPI2: CLK13, DIN14, CS15, DC27, RST26, BUSY25; mode0,1MHz, no MISO.
  GPIO4 is not used: board rail-switch behavior is unverified. Never infer
  physical power removal from a successful Sleep command.
- Each pass is sequential and SHA-256-checked. Refresh is issued only after
  both complete passes match. Duplicate Commit queries completion, never
  repeats a physical refresh. Lost acknowledgment is not automatic success.
- USB preempts network staging, not an in-progress physical refresh.
  Interrupted staging sends Sleep without Refresh; cleanup failure latches
  a hardware fault requiring physical power removal. This interruption path
  still requires hardware qualification.
- Staging idle deadline20s, total120s; bounded driver time is accounted for
  separately. Panel waits use measured BUSY HIGH with10s power-on/off and30s
  refresh budgets, not a fixed delay claiming completion.
- UART logical connection retires after19s idle; owner lease idles at20s.
  Go's explicit ESP32 transport rebinds the existing private claim before new
  work. It never steals another lease or replays an ambiguous transaction.
- RTC retained guard fences software/watchdog reset during panel work.
  **EN reset can report POWERON_RESET while panel rails remain powered**;
  the guard is not proof of a cold boot. Do not reset or reopen serial during
  refresh. After an interrupted/unknown refresh, remove board power before
  trying again.
- `CodeHardware` carries numeric panel code/phase/step/command and observed
  BUSY state. Cleanup preserves the first failure. `--panel-status` reports
  cached evidence, not new GPIO measurements. Storage initialization failure
  keeps USB read-only Inspect available; network/display stay fenced.

## Network and credentials

The board joins WPA2-PSK/CCMP (WPA2/WPA3 transition AP allowed only when CCMP
negotiated). Open/WEP/WPA1/TKIP are rejected; Bluetooth is disabled. Pico's
default WPA3-only provisioning policy is unchanged. This port supports IPv4
and DNS names; IPv6 input is explicitly rejected rather than silently ignored.

The board makes an **outbound** TCP connection to its provisioned manager.
It exposes no HTTP server or internet-facing device listener. EPN2 authenticates
both parties with the USB-generated256-bit device key, then uses directional
AES-256-GCM keys and strictly ordered records. Durable boot epoch plus consumed
connection counter prevents normal-reset nonce reuse. DNS or routing tampering
does not supply the device key. This is the existing project protocol, not TLS
or a claim of independent cryptographic certification.

Keys/Wi-Fi password are installed only through physical USB; never over Wi-Fi.
The host stages a private enrollment and promotes it only after a matching
canonical successful acknowledgment. Ambiguous writes retain pending evidence.
Any provisioning mutation attempt fences the old network key until restart,
even if storage returned an error. Successful mutations acknowledge then reboot.

The last16KiB flash contain two credential sectors plus two epoch sectors.
Do not erase/reinitialize epochs while retaining the same device key. Corrupt
epochs fail closed; recovery requires an explicit fresh-identity/key procedure,
not automatic formatting. Secrets are plaintext at rest in this initial port:
physical flash access/debug access is outside its network security guarantee.
No eFuse/security-boot/flash-encryption settings are burned by this project.
Application/bootloader logs and panic dumps are disabled to protect the binary
UART channel and avoid secret-bearing register/memory dumps. ROM boot messages
may still appear before the app starts.

Wi-Fi uses modem power saving, no heartbeat frames. Reconnect backoff1–64s
plus jitter avoids hot loops. Socket I/O polls cancellation every25ms; USB runs
on a separate task. DNS uses the SDK's finite retry policy and is not immediately
cancellable; the10s TCP connect budget starts after DNS, not before it.

## Build and verify

Inside the already-running project Dev Container:

```sh
. "$IDF_PATH/export.sh"
cd /workspaces/pico-sandbox/experiments/12-esp32-epaper/receiver
idf.py -B build-release -DSDKCONFIG=build-release/sdkconfig build
cmake -S tests -B /tmp/epaper-receiver-tests
cmake --build /tmp/epaper-receiver-tests -j4
ctest --test-dir /tmp/epaper-receiver-tests --output-on-failure
cd tests/interop
EP_RECEIVER_CLI=/tmp/epaper-receiver-tests/receiver_cli \
EP_CRYPTO_CLI=/tmp/epaper-receiver-tests/crypto_cli \
EP_WIRE_CLI=/tmp/epaper-receiver-tests/wire_cli go test -timeout 30s ./...
```

SDK pinned to `b774170ff46c393eeb5e495ea37936038d3f4f4f`; use project Dockerfile,
not an unpinned system SDK. Build host tools from `experiments/03-remote-epaper`
with normal `go build`; cross-compile macOS binaries **inside the container**:

```sh
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o ../12-esp32-epaper/epaperprovision-darwin-arm64 ./cmd/epaperprovision
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o ../12-esp32-epaper/epaperscreen-darwin-arm64 ./cmd/epaperscreen
```

Go↔C tests exercise actual codecs, crypto, lease/stream receiver and panel code
against injected flash/SPI/time. They check two complete frames and provisioning
lifecycle. They do not emulate ESP32 RF, flash-cache concurrency, USB bridge,
panel analogue waveforms or visible pixels.

## Flash layout — do not use the TinyGo flasher

| Offset | Build artifact |
| --- | --- |
| `0x1000` | `build-release/bootloader/bootloader.bin` |
| `0x8000` | `build-release/partition_table/partition-table.bin` |
| `0x10000` | `build-release/epaper_receiver.bin` |
| `0x3fc000..0x3fffff` | Protected credentials/epochs, **not a flash input** |

Use the generated ESP-IDF `flash_args` for the exact build. Official esptool
`write_flash` erases affected sectors, not the whole chip. Do not use
`erase_flash`, a4MiB padded image or the old TinyGo command (which erased the
whole chip). First installation still replaces bootloader/app/partition table;
make an explicit rollback/credential backup decision before flashing.

When this board's serial device is forwarded to the Dev Container, official
SDK tooling is available there:

```sh
idf.py -B build-release -DSDKCONFIG=build-release/sdkconfig -p PORT flash
```

`PORT` must be resolved from the currently connected ESP32, not copied from a
historical Pico port. ESP32 uses BOOT/EN, not Pico BOOTSEL. Host flashing requires
the agreed host tool/permission; this document does not install host Python.

## Direct host USB and provisioning

Source-verified native multi-image command for the existing Go flasher v0.8.1,
run from `experiments/12-esp32-epaper` **only after confirming the current port
and authorization for host USB access**:

```sh
./espflasher-darwin-arm64 -chip esp32 -port /dev/cu.YOUR_ESP32 -baud 115200 \
  -bootloader receiver/build-release/bootloader/bootloader.bin -bootloader-offset 0x1000 \
  -partitions receiver/build-release/partition_table/partition-table.bin -partitions-offset 0x8000 \
  -app receiver/build-release/epaper_receiver.bin -app-offset 0x10000
```

No `-erase-all`; flash headers keep their built DIO/40MHz/4MiB values.
The upstream CLI resets after writing. This is source-verified preparation,
not a claim that the native receiver has been flashed.

Reference: [flasher v0.8.1 CLI](https://github.com/tinygo-org/espflasher/blob/cdcb4bec1d69c043313d21d812822f69a6276d09/main.go).

Use the macOS binaries on the Mac where CH343P is connected. No OrbStack
attach/detach is needed for those binaries. The explicit `esp32:` prefix selects
115200baud and avoids deliberate DTR pulses. **The OS/driver may still pulse
DTR/RTS when opening a port**; this is not a no-reset guarantee.

```sh
./epaperprovision-darwin-arm64 -target esp32 -port /dev/cu.YOUR_ESP32 inspect
./epaperscreen-darwin-arm64 --status esp32:/dev/cu.YOUR_ESP32
./epaperscreen-darwin-arm64 --panel-status esp32:/dev/cu.YOUR_ESP32
./epaperscreen-darwin-arm64 dashboard.html esp32:/dev/cu.YOUR_ESP32
```

USB image delivery does not require Wi-Fi credentials. The command may wait
for the180s safety cooldown. `screen=confirmed` means protocol/controller
completion; verify visible pixels separately.

For Wi-Fi, prepare a private untracked JSON file containing `ssid`, `passphrase`,
`manager` (e.g. `tcp://192.0.2.10:9443`) and `timezone` (default`Europe/Kiev`).
The example IP is documentation-only. Do not paste credentials into chat or
command arguments. Provision generates the device key locally:

```sh
./epaperprovision-darwin-arm64 -target esp32 -port /dev/cu.YOUR_ESP32 \
  -registry /PRIVATE/PATH/device.json provision < /PRIVATE/PATH/wifi.json
```

Use that enrollment with existing Go `epaper-manager -screen
-screen-transport wifi -enrollment ...`. Keep the existing manager TLS/API-token,
trusted-HTML and loopback-only renderer admission rules; this port does not
silently expose untrusted HTML rendering publicly. USB manager mode accepts
`-serial esp32:/dev/cu.YOUR_ESP32 -usb-worker /PATH/epaperscreen-darwin-arm64`.

## Hardware acceptance still required

1. Passed: exact artifact and offsets verified; known-good diagnostic retained.
   Integrated USB fixture `usb-acceptance.html` returned `screen=confirmed`;
   user confirmed `ESP32 USB READY` and the corner time. Exact artifact and
   checkpoint are recorded in `../../../docs/epaper-debugging-history.md`.
2. Still required: repeat cold start; reliable USB Inspect/Hello/Health and two
   distinct visible frames with time. Initial Inspect returned no data; later
   Hello/Health and frame delivery succeeded. The cause remains unknown.
3. Passed: private USB Wi-Fi provisioning, macOS Go manager EPN2 delivery and
   user-confirmed visible `ESP32 WIFI READY` / `TEST FRAME B`. Manager reported
   confirmed1/delivered1 and refresh_trusted=true. No USB frame was sent during
   this Wi-Fi test. Repeated/recovery scenarios below remain separate gates.
4. Passed: graceful manager restart, automatic ESP32 reconnect and a single
   resubmitted scene, with user-confirmed `WIFI RECONNECTED` / `TEST FRAME C`.
   Screen scenes are memory-only: the producer must resubmit HTML after manager
   restart. Still required: wrong key, prolonged disconnect, USB priority and
   lost ACK without replay. This test did not interrupt an active transfer.
5. Passed: idle board power cycle, retained credentials, reconnect to the same
   manager and delivery of fixture D; user visually confirmed it on 2026-09-21.
   Still required: interruption during staging, credential writes or refresh,
   and recovery without repeated refresh after an ambiguous completion.
6. Record real timings, memory high-water marks and power measurements before
   claiming latency/energy improvements or long-running acceptance.

## Sources

- [Board Rev3 schematic](https://files.waveshare.com/wiki/E-Paper-ESP32-Driver-Board/E-Paper_ESP32_Driver_Board_V3.pdf).
- [Pinned exact panel lifecycle](https://github.com/waveshareteam/Pico_ePaper_Code/blob/c9bcd84db5adf5f085353649a8a5c31492bc5fb8/c/lib/e-Paper/EPD_7in5_V2.c).
- [ESP-IDF partition API](https://docs.espressif.com/projects/esp-idf/en/v5.5.5/esp32/api-reference/storage/partition.html),
  [SPI master](https://docs.espressif.com/projects/esp-idf/en/v5.5.5/esp32/api-reference/peripherals/spi_master.html),
  [Wi-Fi](https://docs.espressif.com/projects/esp-idf/en/v5.5.5/esp32/api-reference/network/esp_wifi.html).
- [esptool boot/reset behavior](https://docs.espressif.com/projects/esptool/en/latest/esp32/advanced-topics/boot-mode-selection.html),
  [flash command](https://docs.espressif.com/projects/esptool/en/latest/esp32/esptool/basic-commands.html).
- [Go serial initial modem-bit caveat](https://pkg.go.dev/go.bug.st/serial#Mode).
