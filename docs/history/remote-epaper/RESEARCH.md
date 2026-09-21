# Remote e-paper research

> Superseded source priority (2026-09-01): this file originally treated the raw-panel Good Display sample as normative. The purchased product is a Waveshare 7.5inch e-Paper HAT, product code 13504. Use `../../docs/epaper-code-reuse-research.md` and the current Waveshare `EPD_7in5_V2` implementation before changing the panel driver. Historical measurements below remain diagnostic evidence only.

Verified on 2026-08-30 for Raspberry Pi Pico 2 W, Waveshare e-Paper Driver
HAT Rev2.3, and Good Display GDEY075T7 (UC8179, 800x480, black/white).

This is research evidence, not the approved specification.

## Source precedence

1. The Good Display **GDEY075T7** download page redirects to the current
   panel-specific archive, whose 2024-12-17 sample defines the command sequence.
2. Waveshare documentation defines the Driver HAT wiring, switches, and power
   control.
3. The exact installed TinyGo source defines supported Pico 2 W APIs and their
   buffering behavior.
4. Similar Waveshare 7.5-inch drivers are comparison material only. They must
   not override the panel manufacturer's sequence merely because the resolution
   and controller family look similar.

Primary sources:

- Good Display GDEY075T7 download page (page metadata: 2021-05-26):
  https://www.good-display.com/companyfile/687.html
- The page currently redirects to `A32-GDEY075T7.rar`, containing directory
  `A32-GDEY075T7-FP4G-20241217`. Verified archive SHA-256:
  `9e7e149da6d0a17fa6732f92615517195289e308acc2658d8859aed1226cbe67`.
- Command source inside that archive:
  `GDEY075T7_Arduino/Display_EPD_W21.cpp`, SHA-256
  `03bf7ea9b592bff5cb0fb2014b6adcf59dee7dc445f36659a39289a23f463ad9`.
- Waveshare e-Paper Driver HAT:
  https://www.waveshare.com/wiki/E-Paper_Driver_HAT
- TinyGo Pico 2 W board:
  https://tinygo.org/docs/reference/microcontrollers/boards/pico2-w/
- TinyGo Pico 2 W machine API:
  https://tinygo.org/docs/reference/microcontrollers/machine/pico2-w/
- Waveshare 7.5-inch V2 Python comparison driver:
  https://github.com/waveshareteam/e-Paper/blob/master/RaspberryPi_JetsonNano/python/lib/waveshare_epd/epd7in5_V2.py
- Waveshare Pico MicroPython comparison driver:
  https://github.com/waveshareteam/Pico_ePaper_Code/blob/main/python/Pico-ePaper-7.5.py

## Confirmed stack

- TinyGo 0.41.1, Go 1.26.2, LLVM 20.1.1.
- TinyGo target `pico2-w` supports GPIO, SPI, USB Device, and Wi-Fi.
- `machine.Serial` is USB CDC on this target.
- Wi-Fi dependency selected and source-reviewed for the pending listener:
  `github.com/soypat/cyw43439 v0.1.1`. It is not linked into the fail-closed
  production firmware yet.
- The `cyw43439` blocking network examples require `-scheduler=tasks`.

## Panel driver requirements

The 2024-12-17 Good Display sample uses:

- reset low for 10 ms, then high for 10 ms in the sample; its comments call
  10 ms a minimum, while Waveshare warns that an excessively long reset LOW
  can make a power-switching HAT turn itself off;
- power setting `01 07 07 3f 3f`;
- booster soft-start `06 17 17 28 17`;
- power-on command `04`, 100 ms delay, then wait until BUSY is high;
- panel setting `00 1f`;
- resolution `61 03 20 01 e0`;
- `15 00`, `50 10 07`, and `60 22`;
- old plane `10` filled with 48,000 zero bytes;
- new plane `13` containing exactly 48,000 frame bytes;
- refresh `12`, at least 200 us before polling BUSY;
- deep sleep `50 f7`, `02`, wait for BUSY, then `07 a5`, with no additional
  delay between BUSY becoming idle and command `07` in the source.

The Good Display busy loop reads the BUSY pin directly. It does **not** send
command `0x71`.

The sample's operational constraints must remain explicit in the port:

- never insert or remove the panel FPC while powered;
- initialize again before every full refresh;
- always enter deep sleep after a completed refresh;
- Waveshare recommends at least 180 seconds between full refreshes for panels
  without an approved partial-refresh path;
- after five partial updates, perform a full refresh to reduce ghosting;
- do not expose partial refresh as stable until it passes real-panel tests.

## Conflicts and known defects

The current Waveshare Python/MicroPython comparison drivers still send `0x71`
while polling BUSY. A Waveshare issue reports that removing it avoids one class
of hangs, and Good Display's current panel-specific source omits it:

- https://github.com/waveshareteam/e-Paper/issues/190

The Waveshare driver also uses different power bytes (`28 17`) from the current
GDEY075T7 source (`3f 3f`). The GDEY075T7 values take precedence for this panel.

Partial refresh on related Waveshare 7.5-inch V2 panels has an unresolved
addressing/garbage report. Treat it as experimental:

- https://github.com/waveshareteam/e-Paper/issues/362

The screen FPC orientation was previously inferred incorrectly from a marketing
photo. For the adapter in this experiment, the exposed silver contacts face up,
away from the blue adapter PCB. Physical connector orientation must be verified
from the exact hardware, never inferred from a similar product photo.

## SPI findings

- SPI mode is 0 and bytes are sent MSB first.
- Good Display demonstrates 10 MHz; Waveshare MicroPython demonstrates 4 MHz.
- Start conservatively at 4 MHz, then measure before changing it.
- TinyGo 0.41.1 validates an SDI/MISO pin for RP2350 SPI even though the panel is
  write-only; `machine.NoPin` is not accepted by the RP2 SPI implementation.
- TinyGo 0.41.1 RP2 `SPI.Tx(w, nil)` is the documented write-only form. Its
  installed implementation sends through DMA, waits synchronously for DMA/SPI
  completion, and does not retain `w`; passing the 48,000-byte caller frame
  directly avoids repeated DMA setup without creating another frame.
- The final wiring reserves GP16 for the SPI0 SDI configuration required by
  TinyGo and moves HAT PWR to GP15. The physical GP16-to-GP15 move remains an
  acceptance prerequisite; no pin is multiplexed between SPI and HAT power.
- Software SPI is useful to isolate pin-mux or peripheral problems, but is not
  the preferred final implementation because it keeps the CPU awake longer.

## USB transport findings

TinyGo 0.41.1 implements USB CDC RX and TX with 512-byte ring buffers. Its RX
callback stores only the bytes that fit in the free space. A 48,000-byte frame
therefore cannot be treated as one reliable unframed serial write.

The protocol should:

- use a versioned binary envelope and explicit payload length;
- transfer a full monochrome frame as exactly 48,000 bytes;
- validate dimensions, pixel format, length, sequence number, and CRC before
  touching the panel;
- split input into blocks no larger than 256 bytes;
- acknowledge blocks or use a bounded receive window so the host cannot
  overrun the 512-byte CDC buffer;
- advertise READY only when the device can accept another frame;
- keep diagnostic text off the binary CDC stream.

USB is active when the host opens the CDC interface (`DTR`), not merely when a
cable supplies power. An active USB frame transaction preempts an incomplete
Wi-Fi transaction. Neither transport interrupts a physical e-paper refresh.

TinyGo 0.41.1 target source confirms that `time.Sleep` with a scheduler on
RP2040/RP2350 calls `machineLightSleep`; the RP2 timer implementation arms a
hardware alarm and executes ARM `WFE` for sleeps of at least 10 microseconds.
Therefore the current 5 ms detached and 2 ms attached-idle waits are not CPU
busy loops. They still schedule up to roughly 200 and 500 timer wakeups per
second, respectively, so changing them requires target current and transfer
latency measurements rather than assumption. Inspected files and SHA-256:

```text
runtime/runtime_rp2.go              32bc41e60822c96bb4c73fb2192b50f210a3146f4b378c70620610abb2a7f39d
runtime/scheduler_cooperative.go    0ee07eab991230315ddfc518e2ceb20dd3ea8a640d293859b5f9767f1544d4d7
machine/machine_rp2.go              1bf865710531deb75031988efee48e21c7785b3fabc599659063417b359f6367
machine/machine_rp2_timer.go        85fede846a2cf8542a76e5fb4326c12e49fbb5b53ac2fb251faa3a627219b203
```

OrbStack's current USB documentation distinguishes automatically forwarded
serial/UART devices from dedicated passthrough devices. On the live setup,
Linux bound `2E8A:000A` to `cdc_acm` and exposed `ttyACM0` in sysfs, while the
long-running privileged container lacked `/dev/ttyACM0`. Creating the exact
sysfs-reported `166:0` character node made passive `epaperctl -list` discovery
succeed. This is a container device-node issue, not a protocol or Pico USB-ID
failure. Source: https://docs.orbstack.dev/features/usb

Raspberry Pi documents `picotool -f` only for a device running compatible
code. The live TinyGo 0.41.1 CDC firmware exposed no compatible reset interface,
and target-version source search found no `reset_usb_boot`/Picoboot reset
implementation. Physical BOOTSEL is therefore the verified transition into
flash mode for this setup. Source: https://github.com/raspberrypi/picotool

## Wi-Fi transport findings

The selected `cyw43439` package provides Pico W/Pico 2 W TCP and HTTP server
examples through the heap-conscious `lneto`/`seqs` stack. The examples use
small fixed buffers and `-scheduler=tasks`.

A raw TCP listener carrying the same framed/chunked protocol as USB is the
recommended first implementation:

- one parser and one set of error semantics;
- less code and buffering than a separate HTTP upload implementation;
- easier deterministic USB-over-Wi-Fi arbitration;
- no base64 expansion.

The host client normalizes endpoints with Go's standard `net.SplitHostPort`,
`net.JoinHostPort`, and `net/netip.ParseAddr`, then validates a numeric
1..65535 port. This supports DNS, IPv4, bare/bracketed IPv6, and scoped IPv6
without resolving during parsing, while rejecting empty or ambiguous targets.
Sources: https://pkg.go.dev/net#SplitHostPort and
https://pkg.go.dev/net/netip#ParseAddr

Wi-Fi credentials must not be committed. Network authentication and discovery
remain open specification questions. TLS is not assumed to be affordable until
its flash, RAM, latency, and energy cost are measured on the target.

## Host-side client direction

Image decoding, scaling, rotation, thresholding/dithering, and terminal text
rendering belong in a normal Go client on Linux/macOS, not on the Pico. The
device should receive a ready-to-display 1-bit frame. This saves Pico RAM,
flash, CPU time, wake time, and radio airtime.

A real Linux DRM framebuffer or USB video class is not the first target:
e-paper refresh latency and TinyGo USB complexity make a snapshot client a more
appropriate interface. The client can later provide image, stdin/text, and
terminal-snapshot commands over the same transport.

## Current experimental state

`experiments/02-epaper-smoke-test` contains a panel smoke test. Its latest local
changes remove `0x71` and add a tested software-SPI diagnostic path. That build
has not yet been flashed or validated on the physical screen. Software SPI must
not silently become the production driver without comparative evidence.
# 2026-08-30 measured implementation results

- Protocol resource gate: 400-byte RAM delta and 4,632-byte flash delta against
  the equal-frame baseline; no protocol heap allocation reported by TinyGo.
- Composed USB firmware: 23,596 bytes flash and 54,116 bytes RAM including the
  sole 48,000-byte frame; no allocation reported in `panel`, `protocol`,
  `devruntime`, or `usbtransport`.
- TinyGo 0.41.1 `-print-stacks` emitted identical recursive runtime warnings but
  no numeric maximum for both probes. The gate additionally requires identical
  linked C-stack reservation rather than silently treating missing output as a
  measurement.
- Host serial dependency: `go.bug.st/serial` v1.8.0. Its v1.8.0 enumerator
  disables potentially disruptive active USB probing by default; the client
  uses passive VID/PID enumeration and handles positive short writes itself.
- TinyGo application USB ID inherited by `pico2-w` is `2E8A:000A`; RP2350
  BOOTSEL is distinct (`2E8A:000F`).
