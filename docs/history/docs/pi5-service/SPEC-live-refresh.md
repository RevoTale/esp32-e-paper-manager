# Local V4 live refresh

Status: software implemented on2026-09-08/09. On2026-09-09 the user visually
confirmed partial updates, cleanup after five partials and full recovery after
idle. Measured completion was about0.67s partial and2.47s full. This supersedes
the initial hardware-acceptance gap below; long-run wear/energy remain unknown.

## Decision

The user's Pi 5/Pironman 5 + 2.13-inch HAT Rev2.1/V4 now displays the full
HTML test. SPI5 uses GPIO14/15/16, DC22/RST23/BUSY24. Do not change wiring,
Pironman software, SPI speed or the preserved Pico/7.5-inch firmware.

Use partial sessions by default (user decision2026-09-09): one-second
post-completion interval, configurable but not below one second. Explicit
`-partial=false` retains full-only mode and its 180s floor for rollback.
First explicit submission is immediate in partial mode, not an automatic image
at startup. No shell capture, terminal emulator or automatic polling is implied.

Follow Waveshare V4 Display_Base (both RAM planes + normal full activation),
then Display_Partial (documented reset, setup, new plane, partial activation).
Transfer the complete 4000-byte frame, as upstream does; do not claim region-only
SPI transfer. No custom LUT, overclock or forced temperature value.

At most five consecutive partial cycles, then full on the next changed frame.
Also full on the next changed frame after ten minutes. Both are bounded policy
settings; unchanged content does not drive the panel just to refresh the clock.
The corner clock retains the last full-refresh submission time during partials.
BUSY remains authoritative with the existing finite safety timeout. No overlap,
blind retries, or queued stale frames: retain explicit 429/Retry-After admission.

Keep controller RAM during a short active burst. Deep sleep after 30 seconds of
no physical update and on graceful shutdown; serialize this with frame work.
Sleep or hardware failure invalidates partial base; next frame is full. A process
restart likewise never assumes RAM matches the visible panel. Software cannot
detect an unreported power interruption; no optical-success claim from BUSY.

## Sources and acceptance

- https://files.waveshare.com/upload/4/4e/2.13inch_e-Paper_V4_Specification.pdf
  revision4.0,2023-03-17,p9 recommends full after five partial/fast operations;
  p8 forbids controller commands while BUSY HIGH.
- https://github.com/waveshareteam/e-Paper/blob/a794fbc39656b0f93938d1ffb3fdc77eaed9e9fc/RaspberryPi_JetsonNano/c/lib/e-Paper/EPD_2in13_V4.c
  official HEAD verified2026-09-08; base/partial sequence inspected. Deep sleep
  is at the end of a burst, not between upstream partial updates.

Tests must prove base planes, polarity/padding, every I/O failure stopping,
invalid-state rejection, timeout, partial counter/full deadline, duplicate frame,
idle/shutdown exclusion and failure recovery. Rebuild a separate ARM64 artifact;
preserve `build/epaper-local-full-baseline`. Physical acceptance: base frame,
several different partial frames, cleanup full, idle sleep/wake and Pironman
features intact. Latency, ghosting and energy remain unmeasured until then.

## Software checkpoint

Existing Dev Container35e5044beea1, Go1.26.8, TinyGo0.42.0. Quality task passed
in31s: changed coverage93.8%, total92.0%; live policy and panel driver100%.
Full-module race tests and nativeARM64 executable `-help` passed. Independent
read-only review found no required code defect; its exact-plane/padding test
suggestion was implemented. No checker suppression, skip or new dependency.

Artifact: `experiments/03-remote-epaper/build/epaper-local-live`, LinuxARM64,
CGO disabled,17MiB, SHA256
`0b53161d083f3c70b6c51c32f082d1bc68b641ca0cac677cd1d1bcc6a5a17fcf`
(rebuilt after the default-partial decision; the preceding opt-in build had
SHA256 `6214aba3b5e8f6c5dc801aed856204acaf7d177a6f0d894a948a778507dbb49c`).
Baseline SHA256 remains
`41e1a7d7f4337eb0a44d8d27b93a40002f9cafd2d630d4a581cb8ad1bf5271b5`.
Reports: module `build/live-v4-quality.log`, `build/live-v4-race.log`.
No Pi deployment, commit or push performed.

Default-mode verification: CLI regression tests passed, including explicit
full-only opt-out; focused race tests passed and independent review found no
required issue. All task checks and three TinyGo builds completed, but the task
gate failed its runtime budget (130s versus 90s); no limit was relaxed.
Report: `build/live-default-quality.log`. Hardware partial acceptance remains
pending; the rebuilt executable was not installed on the Pi.
