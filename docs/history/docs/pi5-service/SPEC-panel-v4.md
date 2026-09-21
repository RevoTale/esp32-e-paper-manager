# Spec: panel-v4

Status: approved; portable full-refresh implementation and fault tests added.
Hardware acceptance and Linux integration remain pending.

## Objective and scope

Drive Waveshare 2.13inch e-Paper HAT Rev2.1, V4 monochrome panel with a portable
Go command implementation. Reuse `display.Device`, not the 7.5-inch command
sequence. No Linux, TinyGo machine, HTTP or renderer imports in this module.

Location: `experiments/03-remote-epaper/panelv4/` with adjacent tests.
Toolchain: existing Go 1.26.8 and TinyGo 0.42.0. No dependency required for
controller logic. Existing Pico binaries and `panel/` remain unchanged.

## Contract

- Implement `Capabilities`, `Refresh(display.Frame, display.RefreshMode)` and
  idempotent `Sleep` as specified in `display/surface.go`.
- Native size 122 by 250, stride 16, frame 4000 bytes. Use canonical project
  polarity (0 white, 1 black); invert only at the controller boundary and
  keep six non-visible tail bits per row white. Do not mutate caller storage.
- Accept full refresh only initially. Reject invalid frames and unsupported
  modes before any hardware I/O. Reject overlapping calls rather than race.
- Inject error-returning command/data, reset, BUSY and clock/delay operations.
  Linux GPIO failures must remain observable; do not reuse void GPIO callbacks
  from the older panel as if Linux writes cannot fail.
- One refresh owns reset, initialization, frame upload, activation, BUSY wait
  and deep sleep. BUSY HIGH means busy. No separate PWR signal exists here.
- Use actual elapsed time for bounded waits, not iteration count. Record stage,
  last command, byte offset and last known BUSY level in errors.
- Never activate a partial upload. After a fault, invalidate software state;
  the next accepted update starts a complete reset/init. No automatic retry.
- `Sleep` after successful refresh is an I/O-free no-op. Before the first
  refresh or after a fault it returns `ErrState`: unknown hardware is not
  presumed idle. Only a new complete refresh resets this state.
- Platform adapters own bounded synchronous I/O and a monotonic clock. The
  injected BUSY deadline counts actual elapsed I/O time, not polling count.
- Do not issue sleep commands while BUSY remains HIGH after a timeout. Preserve
  the original failure and report any cleanup failure without claiming sleep.

## Sources and verification

Use official V4 source pinned in `../pi5-2in13-service-investigation.md` and
the linked V4 datasheet. Retrieve the pinned source before porting. Retain
upstream license attribution for a derived command implementation.

Tests: command/data trace, geometry/padding/polarity, malformed input with zero
I/O, reset timing, BUSY initially idle, HIGH-to-LOW, stuck HIGH, I/O failure at
each stage, incomplete upload, cleanup, concurrent call and second refresh.
Fake clock tests must prove deadlines without real sleeps.

Implemented baseline follows pinned C `EPD_2in13_V4.c`: reset HIGH/LOW/HIGH
20/2/20 ms, active-HIGH BUSY polls at up to 10 ms, 10 ms settling after idle,
ordinary 0x24 upload, 0x22/F7 then 0x20 activation, 0x10/01 deep sleep and
100 ms delay. Python uses different post-idle/sleep delays; do not mix paths.
One 16-byte row buffer converts polarity; no second full framebuffer.
No best-effort controller cleanup after an I/O fault: adapter-level CS release
must preserve the original error, but uncertain BUSY makes new commands unsafe.

Planned command from the existing module directory (after implementation):
`go test -race -cover ./panelv4 ./display`.
Add the portable driver to the existing TinyGo build gate. Apply repository
300-line file, 60-line function and 90% changed-coverage constraints.

Done requires software gates plus two visibly distinct full updates on the
real HAT with a completed sleep/reinitialize cycle. Logs alone are insufficient.

Always preserve old drivers. Ask before adding fast/partial refresh, changing
waveforms or testing hardware. Never hot-plug or bypass a BUSY timeout.
