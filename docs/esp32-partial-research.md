# ESP32 7.5 V2 partial-refresh implementation boundary

Status: window encoder, region receiver and candidate SPI lifecycle are native-tested,
not hardware-qualified. Board composition is now available only through the
separate opt-in `CONFIG_EP_EXPERIMENTAL_PARTIAL` candidate. The current accepted
firmware and default recovery build remain full-only. Earlier checkpoints below
describe their own integration state, not a claim of current qualification.

## Verified upstream evidence

Fetched from Waveshare's pinned commit
`c9bcd84db5adf5f085353649a8a5c31492bc5fb8` on2026-09-30:

- [Driver](https://github.com/waveshareteam/Pico_ePaper_Code/blob/c9bcd84db5adf5f085353649a8a5c31492bc5fb8/c/lib/e-Paper/EPD_7in5_V2.c):
  `Init_Part` resets, sets panel mode0x1f, powers on, waits for readiness and
  selects E0=02/E5=6e. `Display_Part` sets50=a9,07, enters91, sets90 window,
  writes13 and refreshes12. It does not sleep after each partial call.
- [Example](https://github.com/waveshareteam/Pico_ePaper_Code/blob/c9bcd84db5adf5f085353649a8a5c31492bc5fb8/c/examples/EPD_7in5_V2_test.c):
  full display precedes Init_Part and a bounded series of window updates.
  Full initialization/clear and Sleep follow the series. The500ms demo delay
  is not a specification permitting indefinite high-rate use.

The dated availability comment in these files directly precedes4Gray, not
Init_Part. Do not misquote it as a proven partial-refresh revision restriction.
The wiki request returned403; its safety guidance was not verified in this pass.

## Current gaps and consequences

The accepted `core/panel75.c` full path performs reset/full initialization for
each begin and deep sleep after commit. Region reception now has its separate
optional `ep_region_sink` and independently hashed old/new passes; see
[EPS2 region contract](eps2-region.md). The default board runtime installs only
the full sink; the separate opt-in candidate also installs the guarded region
adapter. Changing only a capability bit would not complete integration.

Partial series require explicit controller-baseline validity, separate from
confirmed visible pixels. Deep sleep/reset, incomplete staging and ambiguous
refresh must not imply usable retained controller RAM. Source examples do not
prove arbitrary sleep/resume or power-loss recovery.

The upstream end-coordinate encoding subtracts one from the low byte without
borrowing into the high byte. For an exclusive end of256 this differs from
encoding255. Our encoder must subtract before splitting bytes, with regression
tests at256 and512. Determine byte alignment/padding from controller semantics;
do not infer general support from the example's unaligned x=150 alone.

## Ordered work and acceptance

1. Verify controller window/alignment, polarity, old/new-plane and sleep rules
   using exact-panel documentation plus another official implementation.
2. Define a negotiated region transaction, including mode, rectangle, byte
   count and digest. Preserve the existing full-only wire profile unchanged.
   Reject unsupported explicit partial requests instead of silently promoting.
3. Test the panel adapter with a recorded SPI trace: command order, exclusive
   endpoints, sequential chunks, BUSY deadlines, complete-before-refresh and
   abort-without-refresh. Retain bounded scratch RAM, not a new full framebuffer.
4. Integrate receiver validation and Go sender/manager selection. Composite
   overlapping/translucent elements on the server before calculating damage;
   send final packed pixels, not overlapping imperative draw operations.
5. Cover invalid baseline, cut/duplicate/reordered chunks, digest mismatch,
   lost completion, reset, owner changes and full-maintenance promotion in
   native tests and real Go/C interoperability. No blind Commit replay.
6. Qualify the1s requested partial cadence and configurable full policy against
   panel limits. Keep full-refresh timestamp unchanged during partial updates.
   Measure readiness/duration rather than treating a sleep as completion.
7. Run complete quality/resource gates, build a separate candidate, then perform
   bounded physical acceptance with final user visual confirmation. Existing
   accepted full-only firmware remains the recovery checkpoint.

No new waveform or hardware operation was performed during this research.

## Window encoder checkpoint

`core/panel75_window.c` now validates half-open800×480 bounds, requires both X
endpoints divisible by8, returns the packed plane size and encodes command90's
nine data bytes. It neither rounds the request nor touches hardware. Invalid
input leaves both outputs unchanged. The helper is not yet wired into the
device target, sink or capability advertisement.

The native test first failed because the encoder did not exist. After
implementation, native21/21 and the C function-size gate passed. Tests cover
exclusive256/512 endpoints, all5050 valid aligned horizontal intervals, the
last pixel row, full-frame48000-byte size, invalid/empty/reversed/out-of-bounds
rectangles and null outputs. AddressSanitizer/UndefinedBehaviorSanitizer are
enabled. This proves geometry only, not physical partial refresh.

The second official [Python driver](https://github.com/waveshareteam/e-Paper/blob/master/RaspberryPi_JetsonNano/python/lib/waveshare_epd/epd7in5_V2.py)
was read (not executed) on2026-09-30. It confirms subtract-before-split endpoint
encoding. Its alignment expression, buffer allocation and pixel polarity differ
from the Pico example; they are not copied as a complete implementation. Resolve
the canonical on-wire black/white convention and controller lifecycle before
connecting this encoder to the panel adapter.

## Datasheet correction — 2026-09-30

Fact: the [V2 specification](https://files.waveshare.com/upload/6/60/7.5inch_e-Paper_V2_Specification.pdf#page=45)
R90h requires the inclusive end bank and end line to exceed their starts.
The encoder therefore now rejects widths below16 and heights below2. The
earlier8×1 acceptance above is superseded, not a supported hardware promise.
Manager damage expansion must preserve neighboring pixels from the final scene;
the receiver must still reject, never silently expand, incoming rectangles.

Measurement: the new regression failed against the previous encoder; after
correction, native21/21 and `make c-size` passed. The exhaustive horizontal
test now covers4950 valid intervals. Boundary tests retain256/512 borrow,
bottom/right edges and unchanged outputs on rejected inputs.

Fact: the same specification, R10h/R13h and R50h (pages31–32,37–38), defines
old/new SRAM and polarity. With upstream partial setting A9,07, DDX=01 means
both planes use1=white; N2OCP enables copying new to old after refresh.
Inference: our canonical1=black bytes need inversion for both partial planes,
unlike the accepted full path. Do not reuse its pass-dependent inversion.

Fact: R07h requires hardware reset to leave deep sleep; R02h preserves registers
only until deep sleep or power loss. This is not proof of SRAM retention across
sleep. Candidate design: stage authenticated old and new region pixels after
wake, tied to the last confirmed visible-image identity, then sleep after commit.
This avoids relying on retained RAM or keeping high-voltage circuitry powered
between updates. Exact window staging/wake behavior remains to be qualified;
neither the candidate design nor a partial capability is enabled yet.

Verification after the correction and the separately documented manager
cancellation fix: complete `make quality` exited0; local log
`build/quality-partial-and-cancellation.log`. This includes three coverage runs,
native21/21, Go/C interoperability, portable TinyGo compilation, ESP-IDF
application build and Linux manager builds. It is not a new CI or hardware run.

## Candidate panel adapter — 2026-09-30

Fact: `panel75_region.c` implements the pinned Init_Part sequence followed by
50=A9,07;91;90(window). The shared streaming writer sends bounded64-byte SPI
chunks, inverts both partial planes and stops at the validated region byte
count. Commit retains bounded BUSY handling and power-off/deep-sleep; Abort
never issues12. Reset/cycle setup is shared without changing full-path command
order, payloads or delays. The board composition does not enable this adapter.

The pinned local Waveshare source was re-read after remote raw-source retrieval
failed. The official V2 PDF was retrieved again. Restoring OLD/NEW region data
after wake remains a candidate inferred from the register contract, not a
manufacturer-tested sleep-between-partials example or physical acceptance.

Measurement: SPI-model tests verify exact commands, window endpoints crossing256,
different old/new polarity,64-byte scratch bound, complete-before-refresh,
abort/sleep, invalid geometry without I/O and failure at every initialization
and commit I/O step. A failing regression showed the low-level sink could retry
Commit after a driver error; write/commit now reject the latched error. The
receiver already fenced such errors, but the sink no longer depends on callers
to prevent a second refresh. Cleanup remains callable and preserves diagnostics.

Partial diagnostic steps18–21 identify E0,E5,91,90 respectively in initialization
phase1; existing step values remain unchanged. A RED trace test exposed the
previous stale step inherited from full initialization, then passed after mapping.

Measurement: complete `make quality` exited0 (`build/quality-panel-region.log`),
including24 native tests, Go/C interoperability, coverage, TinyGo, ESP-IDF and
Linux builds. A subsequent test-only addition composes the real EPS2 receiver
with this SPI adapter: full800×480 → region32×40 → duplicate Commit. It verifies
48000+160 bytes per plane, correct inversion and exactly two total refresh
commands. Final native/C-size evidence: `build/panel-region-integration.log`.
No new framebuffer was added to device code. Energy and peak resources are
unmeasured; the test's smaller payload is not a measured power improvement.

Remaining: negotiated Go sender/manager, damage/corner/cadence handling and
physical qualification of wake, old/new SRAM, ghosting, faults and timing.
No flash, hardware frame, key write, commit or push was performed.
