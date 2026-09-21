# Spec: linux-panel-io

Status: approved; software adapter implemented, physical acceptance pending.

## Objective

Provide a Linux adapter using kernel SPI and GPIO character devices on Pi 5
RP1. Keep Linux dependencies outside portable display/controller packages.
Proposed location: `experiments/03-remote-epaper/linuxpanel/`.

## Ownership and configuration

- Select the GPIO chip by verified chip/line identity, not a hardcoded
  `/dev/gpiochip4` index. Validate line names/offsets and reject duplicates.
- Request only configured lines; fail if another consumer owns them. No
  sysfs export, direct register mmap, shell-based GPIO writes or root fallback.
- SPI device path, signal lines and frequency are configuration. Mode 0,
  MSB-first and the exact speed must be checked against the HAT reference.
- Give CS exactly one owner. If using kernel CS, portable command/data calls
  map to separate complete SPI transactions; do not also toggle it as GPIO.
- Acquire lines with explicit initial levels in the request; release every
  acquired resource after partial initialization failure.
- Never request a PWR line for this eight-signal HAT.

Candidate: SPI5 transmit-only pinctrl excluding GPIO13, with CS outside
Pironman's occupied pins. This requires an independently reviewed overlay;
the stock overlay is not accepted. Exact assignments remain pending kernel
and schematic validation. Do not publish this candidate as ready to wire.

## Tests and acceptance

Mock the kernel-facing boundary: discovery ambiguity, missing devices,
duplicate/occupied lines, permission failures, SPI short writes, input/output
errors, CS ownership, rollback and idempotent close. Run planned
`go test -race -cover ./linuxpanel` from the existing module after creation.

Before selecting a Go GPIO library, verify maintained Pi 5 character-device
support and license, then pin the dependency. Common panel modules must still
cross-build with TinyGo. Repository quality constraints apply unchanged.

Physical gate: inspect actual Pi kernel and overlay, verify requested lines
only, and retain RGB/OLED/fan operation across service start, refresh and stop.
No successful mock or overlay compile substitutes for this gate.

Ask before modifying homelab boot configuration or installing a service.
Never disable Pironman functions or change the existing Pico implementation.

## Implementation checkpoint — 2026-09-08

- `go-gpiocdev` v0.9.1 (MIT), GPIO character-device ABI v2. An explicit chip
  path is required, but it is not trusted: label must be `pinctrl-rp1` and each
  requested offset must have its matching `GPIO<n>` name and be unused.
  Alias gpiochip4 -> gpiochip0 therefore cannot create ambiguous discovery.
- This Pironman profile permits control inputs/outputs only on GPIO17..27,
  excluding GPIO21 and duplicates. This is a software guard, not a wire table.
- Kernel owns CS. SPI is restricted to `/dev/spidev5.0`, mode 0, MSB first,
  8-bit words, configurable 1..1000000 Hz. Readback must match configuration.
  No probing SPI0 (RGB) or SPI10 (system EEPROM).
- Use normal spidev writes and reject short writes without retry. DC stays
  stable during each write. GPIO acquisition initializes DC LOW, reset HIGH,
  BUSY input, physical active HIGH; startup does not pulse reset or refresh.
- Acquisition errors release all acquired descriptors. Close is idempotent;
  cleanup errors are joined with the original failure, not discarded.
- BUSY uses the portable driver's monotonic deadline. A blocked kernel syscall
  is not interruptible by that deadline; do not describe it as a hard deadline
  for all Linux I/O. Process shutdown/target kernel testing remain service gates.

Sources: [GPIO library pinned source](https://github.com/warthog618/go-gpiocdev/tree/v0.9.1),
[kernel spidev API](https://docs.kernel.org/spi/spidev.html).
