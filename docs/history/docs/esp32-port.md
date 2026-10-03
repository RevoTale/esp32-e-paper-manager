# ESP32 e-paper port

Status: 7.5-inch V2 diagnostic visibly accepted; complete receiver is unfinished.
Date: 2026-09-12.

## Active target and next implementation

The user selected the previous **7.5-inch V2 monochrome 800x480** panel for the
complete ESP32 receiver after confirming its diagnostic rectangle. The small
1.54-inch tricolor profile below remains a preserved, unaccepted alternative.
Do not carry its two-color format or timing into the active target.

Historical work followed [the ESP32 receiver plan](https://github.com/RevoTale/esp32-e-paper-manager/blob/1b60448ddf2fd594bf3e05d3ceafc593f28412c3/docs/history/remote-epaper/tasks/esp32-75.md).
Reuse the native Go manager, EPS2, authenticated EPN2 sessions and exact accepted
7.5 panel sequence. New board adapters must not weaken Pico's security policy.

Fresh source inspection found two integration gates:

- TinyGo 0.42.0 has no original-ESP32 `machine.Flash`; the ESP32-C3 adapter
  cannot be used on this chip. The original ESP32 linker executes code from
  cached flash. A safe persistent adapter needs a separately reviewed layout,
  cache/interrupt handling and verified writes, not just ROM function calls.
- `tinygo.org/x/espradio@v0.3.0/osi.c` NVS setters/commit return success
  without persistence; getters return NOT_FOUND. These are radio shims, not
  a credential store. `radio.c` disables the radio's NVS use. Never expose
  these callbacks as successful application provisioning.

The proposed reserved 16-KiB flash area and new storage backend require approval
under REQUIREMENTS-v2's credential-storage rule. No storage writes, data erase,
new firmware flash or network enrollment occurred during this planning audit.
Existing diagnostic artifacts and the physical assembly remain unchanged.

Primary sources:
[TinyGo ESP32 linker](https://github.com/tinygo-org/tinygo/blob/v0.42.0/targets/esp32.ld),
[espradio NVS shims](https://github.com/tinygo-org/espradio/blob/v0.3.0/osi.c),
[Espressif flash concurrency](https://docs.espressif.com/projects/esp-idf/en/stable/esp32/api-reference/peripherals/spi_flash/spi_flash_concurrency.html).

## Scope and accepted changes

- Preserve all Pico and Raspberry Pi 5 firmware, wiring profiles and artifacts.
- User identifies Waveshare e-Paper ESP32 Driver Board Rev3. Its official
  schematic names ESP32-WROOM-32E and CH343P USB-to-UART, not ESP32-S3.
- User's new panel photo reads E154A79N204Q02 and V2. Its reported tricolor,
  approximately 200x200 geometry matches the Waveshare 1.54-inch (B) V2
  profile. This is a model match, not physical pixel acceptance.
- User explicitly permits C dependencies and WPA2 for this ESP32 port.
  This does not weaken Pico's WPA3 policy, application authentication,
  encryption, replay protection or USB-only provisioning.
- Server-side native Go HTML/CSS rendering remains the authoring path.
  Do not copy the old monochrome controller sequence, partial refresh timing,
  or Pico GPIO mapping to this hardware.

## Verified sources and constraints

1. [Rev3 schematic](https://files.waveshare.com/wiki/E-Paper-ESP32-Driver-Board/E-Paper_ESP32_Driver_Board_V3.pdf):
   MCU and serial bridge identification. Confirm pin nets and switch settings
   from the schematic before any panel operation.
2. [1.54-inch B manual](https://www.waveshare.com/wiki/1.54inch_e-Paper_Module_%28B%29_Manual):
   V2 differs from V1 in controller/driver; 200x200 black/white/red; listed
   full refresh about 14 seconds. No inherited one-second partial promise.
3. [Panel specification](https://files.waveshare.com/upload/9/9e/1.54inch-e-paper-b-v2-specification.pdf).
4. [espradio](https://github.com/tinygo-org/espradio): v0.3.0 downloaded and
   inspected with TinyGo 0.42.0, Go 1.26.8 in the existing Dev Container.
   Original ESP32 is now supported. Its C bridge and Espressif binary radio
   libraries are explicitly permitted by the user.
5. [Historical Espressif TinyGo workshop](https://developer.espressif.com/workshops/tinygo/assignment-5/):
   says original ESP32 Wi-Fi is unsupported; superseded for this investigation
   by current espradio source and the installed TinyGo ESP32 target.

`espradio v0.3.0/radio.c:espradio_sta_set_config` sets the authentication
threshold to WPA2 for nonempty passwords. Do not describe that as WPA3-only
or as proof of the negotiated cipher. Never enable an open network fallback.

## Port gates

1. Compile the pinned upstream ESP32 Wi-Fi example without hardware access.
2. Implement and test the exact two-plane panel lifecycle with bounded BUSY,
   error cleanup and full-only refresh policy. Keep red and black distinct.
3. Adapt board UART/SPI/power separately from portable panel/protocol code.
4. Provide durable identity and USB-only provisioning before enabling secured
   network sessions. The installed original-ESP32 machine package has no
   `machine.Flash` adapter: Pico's flash journal cannot simply be imported.
   Never substitute volatile epochs or a fixed nonce to make networking work.
5. Connect the existing manager and USB protocol with explicit color capability,
   atomic updates, USB priority and no controller operation on invalid input.
6. Build firmware and host sender, run tests/review, then document exact flash
   commands and collect serial plus visible-panel acceptance.

## Flashing boundary

This board uses USB-to-UART, not Pico UF2 mass storage. Installed TinyGo's
ESP32 target selects its `esp32flash` method. No Python tool has been run.
Do not issue a flash/erase command until the exact port and image are verified;
preserve credentials and durable counters across ordinary firmware updates.

## Evidence

On 2026-09-12, `panel154b` implements the vendor's full-refresh two-plane
sequence, input validation before hardware access, bounded BUSY waits, and
stage-specific SPI errors. Its diagnostic accepts an explicit UART `t`, limits
attempt starts to one per 180 seconds, and latches hardware failures. On failure,
CS is released, but panel sleep/power removal is **not guaranteed**: the operator
must disconnect power. Rev3's GPIO4 power-control path includes R35 marked NC
in the schematic, so software power cutoff must not be assumed.

The main module's `scripts/quality.sh task` passed in 25 seconds: changed-line
coverage 92.0%, total 92.1%. These totals include the pre-existing Pi5 worktree
changes. The panel and diagnostic policy have 100% statement coverage; the
ESP32-only machine/UART entry point has no host runtime coverage (0/37 lines,
included in the gate rather than excluded). Existing Pico builds also passed.

The upstream espradio v0.3.0 HTTP example linked for `esp32-coreboard-v2`:
305232 bytes flash and 22416 bytes static RAM. This is a compatibility build
probe, not Wi-Fi, authentication, heap-usage or power-consumption acceptance.

The diagnostic and flashing instructions are in
[experiment 12](../esp32/README.md). No repeated board photo
is required; verify the actual switch labels before first hardware operation.
No `/dev/ttyUSB*` or `/dev/ttyACM*` device was visible in the matching container
at the latest check; this does not establish whether macOS can see the board.

Historical pre-flash boundary: container and installed tool versions were inspected. No board was flashed,
no panel power/refresh command was sent, and no Wi-Fi connection was attempted.
Compilation and physical acceptance are separate gates.
