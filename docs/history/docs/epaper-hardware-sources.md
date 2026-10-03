# E-paper hardware sources

Use these sources when changing or debugging support for the current hardware:

- Raspberry Pi Pico 2 W
- Waveshare 7.5inch e-Paper HAT, product code 13504
- Waveshare e-Paper Driver HAT Rev2.3
- 800 x 480 black-and-white V2 panel

## Source priority

1. [Waveshare 7.5inch e-Paper HAT manual](https://www.waveshare.com/wiki/7.5inch_e-Paper_HAT_Manual) for the product version, display protocol, initialization sequence, refresh behavior, and the `7.5V2` versus `7.5V2_old` distinction.
2. [Waveshare e-Paper Driver HAT documentation](https://www.waveshare.com/wiki/E-Paper_Driver_HAT) for the Rev2.3 connector, `PWR`, voltage, switches, and signal meanings.
3. [Raspberry Pi Pico 2 W documentation](https://www.raspberrypi.com/documentation/microcontrollers/pico-series.html) for GPIO numbers, physical pins, electrical limits, SPI peripherals, and RP2350 behavior.
4. [Waveshare Pico e-Paper example code](https://github.com/waveshareteam/Pico_ePaper_Code) as the upstream Pico software reference. Select the driver for the exact display version; do not infer compatibility from resolution alone.

The current upstream `main` commit is `c9bcd84db5adf5f085353649a8a5c31492bc5fb8`. The unchanged driver copied into `experiments/05-waveshare-official-c/vendor/waveshare/` comes from that exact commit; it is not an outdated fork.

## Required cross-checks

Also compare relevant decisions with both of these requested references:

- [Raspberry Pi Pico and e-paper tutorial by peppe8o](https://peppe8o.com/raspberry-pi-pico-epaper-eink/)
- [Waveshare Pico-ePaper-7.5 Wiki](https://www.waveshare.com/wiki/Pico-ePaper-7.5)

These are cross-checks, not authoritative pin mappings for the current assembly:

- The peppe8o article is a third-party MicroPython tutorial using a 2.13-inch module and different GPIO assignments. Use it for general Pico workflow, framebuffer usage, full refresh, clear, and sleep concepts only.
- `Pico-ePaper-7.5` is official Waveshare documentation for the integrated `Pico-ePaper-7.5` driver board. Our separate e-Paper Driver HAT Rev2.3 has a different connector and an explicit `PWR` signal.
- Never copy GPIO assignments, voltage connections, switch settings, reset timing, or power sequencing from either page without confirming them against the exact HAT revision and Pico 2 W pinout.

## Current implementation rule

Use Waveshare's `7.5V2` implementation as the baseline for the purchased Waveshare kit. Treat panel markings as additional identification, not as a reason to replace the Waveshare product driver automatically. Use another vendor's panel sample only as an explicitly documented diagnostic experiment.

Before changing firmware, record which source confirms each of these items:

- display and HAT revision;
- controller protocol and initialization sequence;
- `PWR`, `RST`, and `BUSY` polarity and timing;
- SPI mode and maximum tested frequency;
- GPIO and physical-pin mapping;
- HAT switch positions;
- full-refresh, sleep, and power-off sequence.

If the sources disagree, stop and document the contradiction before testing hardware.

## Existing code research

See [`epaper-code-reuse-research.md`](epaper-code-reuse-research.md) before writing or replacing display code. It records the available official Pico implementation, TinyGo coverage, relevant third-party Go implementations, compatibility limits, and the recommended reuse boundary.

## Debugging history

See [`epaper-debugging-history.md`](epaper-debugging-history.md) before changing hardware or firmware. It preserves the known-good wiring and switch settings, resolved failures, evidence boundaries, current open issues, and the physical acceptance result.
