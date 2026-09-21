# Spec: Waveshare 7.5-inch V2 panel driver

## Hardware and source authority

- Purchased display: Waveshare product 13504, 7.5-inch black/white V2,
  800 x 480.
- MCU: Raspberry Pi Pico 2 W / RP2350.
- Adapter: Waveshare e-Paper Driver HAT Rev2.3, `Display Config=B` and
  `Interface Config=0` (4-line SPI).
- Official product guide: https://www.waveshare.com/wiki/Pico-ePaper-7.5
- Official source: https://github.com/waveshareteam/Pico_ePaper_Code
- Exact pinned source: `../05-waveshare-official-c/UPSTREAM.md`, driver
  `EPD_7in5_V2.c` V3.0 at commit
  `c9bcd84db5adf5f085353649a8a5c31492bc5fb8`.
- TinyGo target docs:
  https://tinygo.org/docs/reference/microcontrollers/boards/pico2-w/
  https://tinygo.org/docs/reference/microcontrollers/machine/pico2-w/

Similar Good Display, IT8951, 2.9-inch, Linux, and older Waveshare drivers may
inform architecture and diagnostics but cannot override this wire sequence.

## Public contract

`panel.New(IO)` validates callbacks without touching hardware. `Refresh(frame)`
accepts exactly 48,000 bytes, rejects concurrent/reentrant use, performs one
complete full refresh, enters deep sleep, and leaves HAT `PWR` LOW.

Project frame format is row-major, 100 bytes per row, MSB-left, `0=white` and
`1=black`. To match the official Waveshare image polarity, command `0x10`
receives the inverted project frame and `0x13` receives the project frame.
The caller's frame is never modified or retained.

The package performs no automatic retry. A retry policy belongs above the
driver; preserving the first failure is required for diagnosis.

## Verified sequence

1. HAT PWR LOW 100 ms, then HIGH 100 ms.
2. RST HIGH 20 ms, LOW 2 ms, HIGH 20 ms.
3. `01 07 07 3F 3F`.
4. `06 17 17 28 17`.
5. `04`, delay 100 ms, poll BUSY until HIGH.
6. `00 1F`, `61 03 20 01 E0`, `15 00`, `50 10 07`, `60 22`.
7. `10` + 48,000 inverted frame bytes.
8. `13` + 48,000 original frame bytes.
9. `12`, delay 100 ms, poll BUSY until HIGH.
10. `50 F7`, `02`, poll BUSY until HIGH, `07 A5`.
11. HAT PWR LOW.

BUSY is active LOW. The official driver delays first and then waits until BUSY
is HIGH; it does not require observing a separate LOW assertion. Polling is
bounded locally at 10 seconds for power-on, 30 seconds for refresh, and
10 seconds for power-off. These bounds are project safety limits, not Waveshare
timing guarantees.

## Diagnostics and errors

`OpError` contains `Phase`, `Step`, `Command`, `Offset`, optional BUSY level,
and the original cause. `ErrorCode` returns stable codes:

- `E_FRAME_SIZE`
- `E_PANEL_CONFIG`
- `E_PANEL_IN_USE`
- `E_BUSY_TIMEOUT`
- `E_SPI_WRITE`
- `E_PANEL_UNKNOWN`

Optional `IO.Observe(Event)` receives bounded phase, command, transfer progress,
BUSY-wait, and completion events. Production firmware may leave it nil, so
normal refresh has no logging overhead or heap allocation. The observer is
synchronous and diagnostic-only; it must return and must not call `Refresh` on
the same driver.

SPI is write-only. Successful `SPI.Tx` cannot prove that signals reached the
HAT or that the panel accepted data. BUSY is the only controller-originated
software-visible signal; visible output remains hardware acceptance evidence.

## Resource and safety requirements

- No heap allocation in steady-state `Refresh` when `Observe` is nil.
- No second 48,000-byte frame; only one 100-byte inversion row.
- SPI0 mode 0, 4 MHz, SCK GP18, SDO/DIN GP19, unused-required SDI GP16.
- HAT: PWR GP15, CS GP17, DC GP20, RST GP21, BUSY GP22.
- Never change wiring or FPC while powered.
- Ordinary full refresh must complete before any fast/partial/gray mode work.

## Verification

Run inside the existing Dev Container:

```sh
cd /workspaces/pico-sandbox/experiments/03-remote-epaper
go test ./...
go vet ./...
tinygo build -target=pico2-w -scheduler=tasks -o remote-epaper.uf2 ./cmd/device
```

Physical diagnosis uses
`../07-tinygo-waveshare-7in5-v2-diagnostic/diagnostic.uf2` and its USB log.
