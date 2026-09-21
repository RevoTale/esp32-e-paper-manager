# Pico 2 W resource budget

## Accepted baseline

The target is the RP2350A-based Raspberry Pi Pico 2 W. RP2350 provides 520 KiB
(532,480 bytes) of on-chip SRAM. TinyGo documents `pico2-w` support for USB,
SPI, and Wi-Fi. Measurements use the pinned TinyGo 0.41.1 target with the task
scheduler.

The production baseline keeps one 800x480 monochrome framebuffer (48,000
bytes). The HTML update path keeps one 32,768-byte input buffer. Both are
present simultaneously in the full-device differential probe.

## V2-04 tokenizer spike

Measured on 2026-09-02:

| Probe | Flash bytes | Static RAM bytes |
|---|---:|---:|
| current full device | 24,116 | 54,116 |
| full device + 32 KiB input baseline | 24,248 | 86,900 |
| full device + input + tokenizer | 26,320 | 86,924 |
| tokenizer differential | 2,072 | 24 |
| full device + input + parser nodes | 30,164 | 97,268 |
| parser differential | 5,916 | 10,368 |
| full device + parser + CSS storage | 35,972 | 102,972 |
| parser/CSS differential | 11,724 | 16,072 |
| full device + input + parse/CSS/layout/raster storage | 49,028 | 105,340 |
| full render differential over input baseline | 24,780 | 18,440 |

The isolated tokenizer differential is 2,136 flash bytes and 24 static RAM
bytes. Host tests process the adversarial 32 KiB `<b>x</b>` stream with zero
heap allocations, 32,768 forward-consumption work units, and deterministic
EOF. The input is borrowed; tokens are views into it.

After the full render probe, 427,140 bytes of physical SRAM remain.
That is sufficient to proceed with bounded parser, style, layout, crypto, and
network structures, but it is not permission to spend the remainder. Every
later phase must repeat the differential target measurement. The current
device baseline links the existing CYW43439/lneto path; WPA3-only behavior and
the final encrypted link are not yet accepted.

TinyGo reports the same runtime-recursive stack roots in both isolated probes,
so it does not yield a finite maximum stack number. This is recorded as an
instrumentation limitation, not reported as zero stack use. No parser
recursion is allowed; depth is explicit and bounded.

## Final V2 software build

Measured after the full V2 gate on 2026-09-02:

| Artifact | Flash bytes | Static RAM bytes |
|---|---:|---:|
| USB firmware | 237,900 | 105,564 |
| WPA3 firmware | 700,320 | 112,156 |

The complete render resource probe uses 156,580 static RAM bytes when its
32 KiB input, parser, CSS, layout, raster workspace, and 48,000-byte framebuffer
are all retained. The gate limit is 160,000 bytes. These are compiler/linker
measurements, not physical current, stack-watermark, or battery evidence.

## Gates

`scripts/check-html-resources.sh` fails if the tokenizer adds heap allocation,
more than 2 KiB measured RAM, or more than 16 KiB flash. It stores the raw size,
stack, and allocation reports under `resource-reports/html/`. Full-device
baseline and tagged-probe totals must also remain visible in `result.txt`. The
complete render probe fails above 160,000 static RAM bytes.

## Sources

- RP2350 SRAM and memory architecture:
  https://www.raspberrypi.com/documentation/microcontrollers/microcontroller-chips.html
- RP2350 datasheet:
  https://datasheets.raspberrypi.com/rp2350/rp2350-datasheet.pdf
- TinyGo Pico 2 W target support:
  https://tinygo.org/docs/reference/microcontrollers/boards/pico2-w/
