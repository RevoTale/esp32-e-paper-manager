# E-paper code reuse research

Verified on 2026-09-01 for Raspberry Pi Pico 2 W, Waveshare 7.5inch e-Paper HAT product 13504, e-Paper Driver HAT Rev2.3, and the 800 x 480 black-and-white V2 panel.

## Result

Do not rewrite the complete firmware. Keep the existing TinyGo transport, protocol, security, runtime, host client, and frame-conversion code. Replace only the panel command implementation with a direct, auditable port of Waveshare's current `EPD_7in5_V2` code while retaining the existing TinyGo `machine.SPI` and GPIO adapter.

The official C implementation is small and already separates controller commands from platform I/O through `DEV_Config`. A line-for-line Go port of that controller layer is lower risk than importing a Linux driver or mixing Pico SDK hardware ownership into the TinyGo runtime.

## Official implementations

### Waveshare Pico code

- Repository: https://github.com/waveshareteam/Pico_ePaper_Code
- Current `main` commit verified with `git ls-remote`: `c9bcd84db5adf5f085353649a8a5c31492bc5fb8`
- Driver: `c/lib/e-Paper/EPD_7in5_V2.c` and `EPD_7in5_V2.h`
- Local unchanged copy: `experiments/05-waveshare-official-c/vendor/waveshare/`
- Local Pico 2 W HAL and PWR adaptation: `experiments/05-waveshare-official-c/DEV_Config.c` and `main.c`

This is ready controller code for Waveshare's Pico ecosystem. Only the GPIO, SPI, delay, BUSY, and HAT `PWR` adapter is platform-specific.

### Waveshare general e-Paper code

- Repository: https://github.com/waveshareteam/e-Paper
- Current monochrome V2 driver: https://github.com/waveshareteam/e-Paper/blob/master/RaspberryPi_JetsonNano/c/lib/e-Paper/EPD_7in5_V2.c
- Current test flow: https://github.com/waveshareteam/e-Paper/blob/master/RaspberryPi_JetsonNano/c/examples/EPD_7in5_V2_test.c
- Panel specification: https://files.waveshare.com/upload/6/60/7.5inch_e-Paper_V2_Specification.pdf

Use these as a second official comparison for initialization, full refresh, BUSY handling, and sleep. Do not copy Raspberry Pi Linux GPIO numbers into Pico firmware.

## TinyGo support

- TinyGo driver model: https://tinygo.org/docs/concepts/drivers/
- Official device list: https://tinygo.org/docs/reference/devices/
- Driver repository: https://github.com/tinygo-org/drivers
- Current `release` commit verified with `git ls-remote`: `8f372935ac8b1856c545180824030458b216c88f`

The official TinyGo driver repository has Waveshare drivers for 1.54, 2.13, 2.66, 2.9, and 4.2-inch panels. It does not contain a 7.5-inch V2 driver. There is therefore no official TinyGo package that can be imported for this exact panel.

TinyGo supports CGo, but direct reuse of the Waveshare C files would still require a custom HAL and a carefully controlled C build. The official Pico code expects Pico SDK functions, while the current application owns SPI, GPIO, timing, USB, and Wi-Fi through TinyGo. Treat CGo as a possible fallback, not the default integration.

## Relevant third-party Go code

### `ditrytus/epd7in5v2`

- Repository: https://github.com/ditrytus/epd7in5v2
- Current HEAD verified with `git ls-remote`: `edf327a3a65134ab1b42cc87d2de0afe54e4c805`
- Target: Waveshare 7.5-inch V2 black-and-white panel with Driver HAT Rev2.3
- Limitation: uses `periph.io` Linux GPIO and SPI host drivers, not TinyGo `machine` APIs; its API is young and not frozen.

This is useful for reviewing command encoding, validation, timeout behavior, and known vendor-code defects. It cannot be imported unchanged into Pico 2 W firmware.

### `mxcu/go-waveshare-7.5-epd-v2`

- Repository: https://github.com/mxcu/go-waveshare-7.5-epd-v2
- Current HEAD verified with `git ls-remote`: `b448bb7d703ea3f1f07d473873016f70aac6de7e`

This is another independent Go comparison implementation. Its hardware and TinyGo compatibility have not been accepted for this project; do not add it as a dependency without a source and build audit.

### `nii236/go-waveshare-epaper`

- Repository: https://github.com/nii236/go-waveshare-epaper
- Current HEAD: `ad9ddcc82a89f99e9efde2a712138bb267aabf94`
- Last commit: 2020-12-17
- Status: rejected as an implementation dependency.

The repository is a useful historical comparison but is not usable for this firmware:

- `epd/rpi_driver.go` depends on the Linux Raspberry Pi `github.com/kidoman/embd` host and cannot use TinyGo `machine` on Pico 2 W;
- it has no Driver HAT Rev2.3 `PWR` handling;
- its BUSY loop repeatedly sends command `0x71`, unlike the current Waveshare Pico driver;
- its initialization predates the current Waveshare V3.0 source and omits the current `0x06` booster-soft-start sequence;
- `epd/waveshare_driver.go` contains unfinished or invalid Go, including an empty `GetBuffer` body and invalid `Pixel` construction, so the package is not in a buildable ready state;
- the imported `github.com/MaxHalford/halfgone` package is absent from its `go.mod`;
- the repository contains no license file, so its code must not be copied into this project.

Do not port from this repository. Prefer the current official Waveshare source and use this repository only to understand older approaches that must not be restored.

### `riyaz-ali/epd`

- Repository: https://github.com/riyaz-ali/epd
- Current HEAD: `84f4fd4927b991e28bb7e960f346a820c919adee`
- Last commit: 2020-09-21
- Target: Waveshare 2.9-inch, 296 x 128
- License: MIT
- Status: rejected as a panel implementation; useful only as an interface-design comparison.

It cannot drive this hardware:

- its controller commands, LUT, RAM windows, and refresh sequence target a different 2.9-inch controller, not the 7.5-inch V2 UC8179/GD7965 family;
- the 296 x 128 geometry is coupled to that command implementation, so changing width and height is insufficient;
- it has no Driver HAT Rev2.3 `PWR` handling;
- its example depends on Linux Raspberry Pi `go-rpio`, not TinyGo `machine` on Pico 2 W.

Its narrow `WriteablePin`, `ReadablePin`, and `Transmit` interfaces are a reasonable design. The current `panel.IO` already provides the equivalent boundary with error propagation and bulk writes, so do not add this dependency or port its controller sequence.

### Reddit: 7.8-inch IT8951 HAT on Raspberry Pi Zero 2 W

- Discussion: https://www.reddit.com/r/golang/comments/1jivro4/programmin_waveshare_78_epaper_hat_on_raspberry/
- Product discussed: https://www.waveshare.com/wiki/7.8inch_e-Paper_HAT
- Go package discussed: https://pkg.go.dev/github.com/peergum/IT8951-go
- Status: not applicable as a driver; relevant as an engineering comparison.

The post concerns Waveshare's 7.8-inch HAT with an IT8951 controller connected to a Linux Raspberry Pi Zero 2 W. Our hardware is the 7.5-inch V2 800 x 480 panel connected through e-Paper Driver HAT Rev2.3 to a Pico 2 W. IT8951 has a different command protocol, display memory model, and host interface, so neither the linked Go package nor the USB programming guide can drive our panel.

The package also exposes Raspberry Pi GPIO numbers and Linux-oriented SPI setup rather than TinyGo `machine` APIs. Its GPL-3.0 license would additionally prevent copying code into this project without accepting GPL obligations.

Transferable lessons include separation of transport from controller logic, image preparation, error recovery, repeated-refresh testing, and the complete init/refresh/sleep lifecycle. Use these as design and diagnostic input while keeping the vendor's verified power, refresh, BUSY, and sleep sequence authoritative. Treat claims in the comments about overheating, capacitor discharge, and recovery times as hypotheses to test, not specifications.

## TinyGo port status

Implemented on 2026-09-01 in
`experiments/03-remote-epaper/panel/driver.go` and verified by host tests:

- command order and payloads now follow pinned Waveshare `EPD_7in5_V2.c` V3.0;
- reset is HIGH 20 ms, LOW 2 ms, HIGH 20 ms;
- power-on and refresh wait 100 ms before polling only for BUSY HIGH;
- the driver no longer rejects a valid immediate-HIGH BUSY result or hides the
  first failure behind an automatic retry;
- project frame polarity (`0=white`, `1=black`) is explicitly adapted to
  Waveshare: `0x10` receives the inverted frame and `0x13` the original frame;
- errors now include phase, named step, command, plane byte offset, and known
  BUSY state;
- an optional bounded event observer supports diagnostics while normal refresh
  remains allocation-free and keeps only a 100-byte inversion row.

The separate
`experiments/07-tinygo-waveshare-7in5-v2-diagnostic` firmware waits for USB CDC
DTR, reports every observable stage, samples BUSY during long fixed delays, and
repeats the final result after reconnect. It does not start Wi-Fi.

## Reuse boundary

Keep:

- `protocol`, `securetransport`, `usbtransport`, `wifitransport`, and `devruntime`;
- the 48,000-byte frame format and host-side image conversion;
- TinyGo `machine.SPI0` and GPIO composition;
- bounded errors and tests, provided they do not change the verified wire sequence.

Replaced and revalidated:

- `panel/driver.go` initialization, reset, BUSY, display-plane, and sleep sequence;
- panel-driver tests and source citations;
- `SPEC-panel-driver.md` source priority and lifecycle contract.

Still required:

- transfer the reset-release BUSY precondition proven by the C control
  experiment into the TinyGo driver and its tests;
- run the TinyGo firmware on the physical panel and preserve its USB output;
- correct and visibly verify frame polarity before remote/Wi-Fi testing;
- update remaining historical wording only where it still claims to describe
  current behavior.

Do not adopt partial refresh, fast refresh, or four-gray mode until ordinary full refresh succeeds on the physical panel with the official sequence.

## Physical acceptance update

The unchanged official Waveshare C diagnostic in
`experiments/05-waveshare-official-c` previously completed with BUSY reading
high but produced no visible image. That result remains evidence only for that
specific run; it did not prove pixel output.

After correcting the shifted GPIO wiring and adding a BUSY-idle precondition
after hardware reset, `experiments/11-gdey075t7-smiley` completed the full
init, 96,000-byte frame transfer, refresh, power-off, and deep-sleep lifecycle.
USB reported `RESULT=DONE`, and the user visibly confirmed the smiley. This
proves the Pico 2 W, HAT, adapter, FPC, panel, power path, and command path can
work together. The visible background was black, so frame polarity remains an
open software issue. See `epaper-debugging-history.md` for the evidence and
regression guards.

## Native dashboard rendering decision

Accepted on 2026-09-02.

### Rendering boundary

Do not render general HTML/CSS or run Ebitengine on the Pico 2 W. An exact
Glance dashboard remains a browser-rendering problem and must be converted on
the server into the panel's 800 x 480 1-bit frame format. The Pico should only
authenticate, receive, validate, display, and retain that prepared frame.

If exact Glance appearance is not required, a native Pico dashboard may instead
receive a small, purpose-built data model and draw it locally. Use:

- TinyDraw for lines, rectangles, circles, and triangles:
  https://pkg.go.dev/tinygo.org/x/tinydraw;
- TinyFont for text: https://pkg.go.dev/tinygo.org/x/tinyfont;
- the TinyGo `drivers.Displayer` boundary: `Size`, `SetPixel`, and `Display`.

Implement a small adapter over the project's existing 1-bit framebuffer.
`SetPixel` must own the black/white polarity conversion, and `Display` must
perform one complete panel lifecycle after all drawing is finished. Do not
refresh from individual drawing operations.

TinyDraw is experimental and documents that it is not optimized for speed or
memory. Keep it behind the adapter, pin the reviewed version, run a Pico 2 W
TinyGo build, and measure the final binary and runtime memory before accepting
it for the remote firmware.

### Memory consequence

One 800 x 480 1-bit frame is 48,000 bytes. Reuse that frame when writing the
controller planes sequentially; do not allocate a second full-size plane merely
because the controller accepts two plane commands. The framebuffer itself does
not "overload" RAM, but total acceptance must include the Wi-Fi stack, protocol
buffers, TinyDraw/TinyFont, stack, and observed heap headroom.

### Refresh strategy

Use full refresh first. Partial refresh is not a RAM-protection mechanism: it
primarily reduces refresh latency, visible flashing, data transfer, and refresh
energy for a bounded changed region. Depending on the implementation, detecting
or restoring changed regions may require additional old-frame or tile state.

Waveshare documents partial and fast refresh for newer 7.5-inch V2 panels and
warns that repeated partial refreshes require periodic full refresh to prevent
increasing residual images and possible abnormal display behavior:
https://www.waveshare.com/wiki/7.5inch_e-Paper_HAT_Manual. The official example
performs ten partial time updates before leaving the partial-refresh loop:
https://github.com/waveshareteam/e-Paper/blob/86aa9932f471a50157cf02fafadc2c1b4a965449/RaspberryPi_JetsonNano/c/examples/EPD_7in5_V2_test.c.

Do not infer support solely from the product family name. Add partial refresh as
a separate experiment only after matching the exact panel revision to the
current Waveshare sequence and preserving a known-good full-refresh recovery.
The initial project policy is conservative: no more than five consecutive
partial updates before a full refresh. This is a project guard, not a claimed
manufacturer limit, and must be revised from physical ghosting and power
measurements. After either refresh mode, complete sleep and drive HAT `PWR` low.
