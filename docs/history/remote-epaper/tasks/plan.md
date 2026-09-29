# Current plan: pure-Go manager engine and unified streaming firmware

Updated 2026-09-06 after the user's explicit migration approval. Execute
[the pure-Go migration task list](pure-go-engine.md) under
[ADR-013](../../../../decisions/013-pure-go-render-engine.md). It supersedes the Blitz
choice below, carries forward uncompleted transport/security/timestamp work,
and defers manual hardware testing until the candidate firmware is ready.

E1/E2 update, 2026-09-07: the exact 67-file Rust/IPC source removal is recorded
in [the preservation map](../../../blitz-removal-map.md). The old renderer paths
and commands below are historical checkpoint instructions; active host entry
points use the native Go engine. Ignored artifacts and physical experiments
remain preserved. No container rebuild or physical acceptance is implied.
Checkpoint before migration: `37a1471`. Do not confuse a plan with completion.

# Historical plan: server-rendered streaming path

Updated 2026-09-06. This section supersedes the historical v2/v1 plans below.
Those plans describe earlier firmware, not completion of the new architecture.

## Accepted direction and verified checkpoint

- TinyGo Pico 2 W; HTML/CSS layout and compositing run on the manager with Blitz.
  Pico receives resolved pixels/operations, not a new browser or HTML parser.
- Controller-RAM streaming remains an isolated candidate. EPS1 full-frame USB
  passed visible S1–S4 tests, repeat delivery, and tested first-/second-plane
  interruption preservation followed by recovery without agent-triggered reboot.
- Latest full manager-path checkpoint: S5, 2026-09-06. One HTML scene passed through the
  localhost HTTPS manager → real Blitz → direct USB → Pico → panel, terminal
  `confirmed=1` followed by user-confirmed `MANAGER S5` and `ТЕСТ S5 · 06.09.2026`.
  Same firmware/wiring/180s guard; manager stopped after this single update.
- Existing buffered firmware is retained for recovery. EPS1 is USB-only and
  unauthenticated, not the final manager protocol or production release.
- Latest physical control: inspected S6 BZM1 replay, user-confirmed 2026-09-06,
  following successful half-black/half-white raw control. Same S6 USB client
  and firmware; rendering completed before USB opened. Keep the immutable
  bitmap for comparisons with live delivery; do not infer a wiring/Blitz fault.
- S6 was sent once after explicit host permission: two embedded PNG copies,
  checked BZR4, static-content gate and fixed test badge. Terminal status reached
  `confirmed=1` at 16:42:59 UTC; manager stopped cleanly. User reported S5 still
  visible: stop new feature work and diagnose this acceptance failure. S5 remains
  the accepted live-manager checkpoint, while the later raw S6 replay passed. See
  `../docs/screen-manager-usb.md` for artifacts and acceptance.
- WPA3-only and authenticated/encrypted manager links remain mandatory for
  network delivery. Native Pico IPv6 is optional later; manager IPv6 is separate.
- Server owns cadence and optional debounce/max_wait. One canonical target per
  transaction; Pico waits for the server. Full-maintenance policy defaults to
  600 seconds, configurable; experimental EPS1 currently retains its 180-second
  minimum full-refresh guard. Do not silently change hardware policies.
- Timestamp corner denotes the start of the last successful full cycle; partial
  updates preserve it. Current S1–S5 badges are test labels, not that feature.

## Ordered remaining increments

**Approved deferral, 2026-09-06:** checked CSS/worker diagnostics and manager
recovery are integrated. An intervening static wrapper captures absolute
positioning in Blitz. The user chose no engine fix and an exact-test skip;
BLITZ-POS-001 now permits that reasoned ignore with original assertions retained.
This supersedes the earlier stop, not the required positioning semantics.
Continue independent profile work without a renderer fork or a full-conformance
claim. Reproduction and expiry: `../docs/blitz-positioning-gap.md`.

1. **Server coordinator.** `renderbatch` grouping/lease policy now composes with
   `manager.Screen`: copied full scenes, stale base/render rejection and matching
   delivery results. Real Blitz preview passed. Opt-in loopback HTTPS scene API,
   Screen-level pump and supervised direct USB worker are implemented and
   host-tested; see `../docs/screen-manager-usb.md`. On 2026-09-06 direct macOS
   manager S5 reached terminal `confirmed=1`, then the user confirmed its visible
   output. Public renderer-input gates, device epoch/resync and confirmed pixel
   baseline remain. Keep one
   active and one newest pending target; never merge independent patches.
2. **Renderer/profile integration.** Reuse `blitzworker` and `renderdiff`;
   finish viewport/assets/CSS fixtures and protected timestamp composition.
   Verify actual locked renderer output, not only its advertised capabilities.
   `rasterasset` validates bounded PNG data URLs. Approved `x/net v0.58.0` now
   supplies HTML parsing, whole-scene budgets and scene-local asset IDs; Go
   race/fuzz checks pass. BZR2 encoder/decoder and raw Blitz resource injection
   are implemented, but two real image-layout tests fail on both Blitz beta.1
   and the approved beta.2 dependency set. PR #605 did not fix these fixtures.
   The user clarified: keep images enabled and skip only these two tests.
   Executable BZR2 now works; exact image pixels and real Go → Blitz preview
   pass. The root image exception allows only those two reasoned ignores; original
   assertions stay intact and bugs remain open. No vendor patch is applied.
   See `../docs/blitz-image-layout-bugs.md` for reproductions and exception expiry.
   Ten real-worker viewport/basic box fixtures now pass, including nested
   percentages, positioning, clipping and inline alignment. Go → Blitz preview
   has readable Ukrainian text; see `../docs/blitz-viewport-css.md`.
   `refreshstamp` now provides isolated full-cycle confirmation/partial-label
   state and bounded corner composition; race tests and real stamped PNG pass.
   See `../docs/refresh-timestamp.md`. Live integration still needs trusted
   physical start/completion metadata, provisioned timezone binding and reserved
   layout diagnostics; no S5 timestamp or maintenance scheduler is claimed.
   Nine real-font text fixtures and four cascade fixtures now pass, including
   wrapping, clipped glyphs, inherited style and precedence. Ellipsis painting
   and pre-line/break-spaces semantics remain gaps; no new ignores. See
   `../docs/blitz-text-css.md`. `csscheck` now validates declaration
   grammar using locked cssparser/Stylo with strict recovery rejection, typed
   byte spans and shared input/token/depth/declaration limits (21 CSS tests).
   Canonical HTML discovery and checked BZR3/BZE1 worker handoff are integrated;
   six wire/process tests prove typed rejection before rendering and image output.
   The manager preserves the confirmed scene on rejection, reports its revision
   and location, and waits for corrected input; fatal stale failures still stop.
   After the approved positioning deferral, an isolated native declaration core
   now validates canonical property IDs, shorthand outputs and specified-value
   bounds; twelve new tests include exact real pixels and BZR3 compatibility.
   Native selector admission is now implemented with eight added tests, including
   real cascade output and an unscoped-style fallback boundary reproduction.
   Actual-target ownership is now a bounded opt-in `domscope::SceneScope` using
   the Blitz DOM/matcher, with mixed-owner, HTML-repair, marker and budget tests.
   See `../docs/css-target-ownership.md` for the exact contract and pending gates.
   Opt-in `styles()` now binds actual-DOM stylesheet/inline declarations with
   exact source spans, shared target lists, bounded discovery and all-or-nothing
   diagnostics. Fourteen new tests include same-document cascade/pixel equality.
   See `../docs/css-style-bindings.md`. Checked Go/Blitz source identity now uses
   BZR4 for every Go scene, including empty manifests; fourteen added Rust tests
   and real Go previews pass. See `../docs/css-source-identity.md`.
   Per-target fallback authorization remains open. Keep Default/BZR4 policy
   grammar-only until checked ownership integration,
   broad shorthand resets, font/background assets and computed bounds compose.
   See `../docs/native-css-declarations.md`. Public-input isolation stays open;
   see `../docs/css-declaration-validation.md` and `../docs/blitz-checked-ipc.md`.
   The Go static-content gate now rejects scripts/forms/active attributes before
   image decoding and worker launch, even in bitmap blocks; tests/review pass.
   See `../docs/static-html-admission.md`. Full input isolation remains open.
   Missing-glyph/general-
   overflow and timestamp layout diagnostics stay open.
   Timestamp transport integration remains gated on the final screen contract.
   See `../docs/manager-image-assets.md` for image limits.
   Physical image delivery, broader CSS/codec profile and resource
   measurements remain open. No S5 firmware/refresh change.
3. **Final screen transport.** Resolve the EPS1 prototype versus command draft,
   negotiate only proven operations, measure total wire bytes and peak RAM.
   Keep authenticated chunk validation ahead of SPI and refresh behind Commit.
4. **Secure network composition.** Reuse provisioning/security modules but bind
   them to the new screen path with USB arbitration. Accept WPA3-only IPv4 and
   reconnection on hardware; do not expose raw EPS1 publicly.
5. **Panel policy/optimization.** Exact-panel partial-refresh experiment,
   configurable bounds and maintenance scheduling; measure heap/stack, latency,
   powered time and energy before claiming optimization.
6. **Final acceptance.** Remaining interruption points, lost replies, power loss,
   repeated updates, security review, reproducible tools and recovery docs.

References: `../docs/render-design-review-2026-09-05.md`,
`../docs/render-conflicts-and-module-boundaries.md`,
`../docs/controller-ram-streaming.md`, `../docs/eps1-usb-streaming.md`.
No commit or push without the user's explicit request.

# Historical plan: Remote HTML e-paper v2

Status: software implementation is complete through WPA3 IPv4 compilation and
manager dual-stack support. The task gate passes. Physical USB/WPA3/energy
acceptance remains, and Pico target IPv6 is blocked by the embedded stack's
missing complete SLAAC/router lifecycle. The incomplete v1 plan is preserved
below and its remaining acceptance work is explicitly carried into v2.

## Outcome

Ship a generic TinyGo UF2 for Raspberry Pi Pico 2 W that receives bounded HTML
preferentially over USB or through a Pico-initiated encrypted manager link,
renders it locally into a reusable monochrome display abstraction, and performs
safe full refreshes on the verified Waveshare 7.5-inch panel.

## Fixed Decisions

- The Pico is never exposed as a public inbound service. A home manager exposes
  HTTPS/VPN access and the Pico initiates its encrypted connection.
- Wi-Fi is WPA3-only; unsupported networks fail closed without WPA2 downgrade.
- SSID, password, manager address, device key, and timezone are provisioned only
  over physical USB and are not embedded in the generic UF2.
- USB update traffic has priority, but no transport may interrupt an active
  physical panel refresh.
- HTML/CSS support is an explicit bounded profile, not a browser. Input is at
  most 32 KiB and overflow produces typed, visible diagnostics.
- Rendering uses one 1-bit framebuffer. The update timestamp occupies a
  reserved bottom-right rectangle.
- Full refresh is the production baseline. Partial refresh and Bluetooth are
  out of scope until separately researched and accepted.
- Work is bounded structurally. Five seconds is the render target; 15 seconds
  is a stall watchdog, not a normal rendering deadline. Panel timing is tracked
  separately.
- New/changed Go code targets at most 300 lines per file, 60 lines per function,
  cyclomatic complexity 10, changed-line coverage at least 90%, and no total
  coverage regression.

## Delivery Phases

### Phase A: Reproducible quality and contracts

1. [complete] Add a separate pinned Go tools module and deterministic
   fast/task/full gates.
2. Freeze display, document, update, storage, clock, and transport contracts.
3. Add reusable fakes and resource probes that compile for `pico2-w`.

Checkpoint A: host tests and lint pass, TinyGo compiles, current technical debt
is reported rather than hidden, and firmware dependencies are unchanged.

### Phase B: USB-first HTML vertical slice

4. Implement the bounded HTML tokenizer/parser and CSS subset.
5. Implement layout, overflow policy, timestamp cutout, and 1-bit rasterization.
6. Join them in a budgeted render pipeline using the sanitized Glance fixture.
7. Adapt the existing USB/runtime path and known-good panel driver.

Checkpoint B: the sanitized HTML fixture sent over USB produces the expected
golden frame and a visible full refresh on the physical 7.5-inch display.

### Phase C: Provisioning and remote manager

8. Implement atomic USB-only provisioning, rotation, recovery, and factory reset.
9. Implement the host provisioning utility and generic-UF2 operator workflow.
10. Implement the manager HTTPS API, bounded update retention, and status model.
11. Implement the Pico-initiated authenticated encrypted device link.

Checkpoint C: secrets never cross the public API, replay/tamper tests fail
closed, USB still preempts incomplete network work, and recovery requires USB.

### Phase D: Network and device acceptance

12. Accept WPA3-only IPv4 on hardware, including reconnect and power cycling.
13. Accept IPv6 and public-manager routing without inbound Pico port forwarding.
14. Measure RAM, flash, stack, rendering, radio time, refresh spacing, and sleep.

Checkpoint D: network, energy, longevity, and recovery evidence meets the
accepted requirements on the real Pico 2 W and panel.

### Phase E: Product completion

15. Complete operator/developer documentation and the evidence matrix.
16. Repeat test, review, security, simplification, and fix loops until all gates
    pass without waivers or weakened constraints.

Checkpoint E: every requirement links to automated or physical evidence; the
generic UF2, host tools, manager, and recovery instructions are reproducible.

## Dependency Order

`quality/contracts -> parser -> layout/raster -> USB physical slice ->
provisioning -> manager/device link -> WPA3/IPv6 -> final acceptance`

No network implementation blocks the first useful HTML-on-panel result.

## v1 Carry-over

- v1 Task 5 is absorbed by v2 physical panel acceptance.
- v1 Task 7 recovery cases and the USB end-to-end checkpoint are absorbed by
  the USB vertical slice and final physical matrix.
- v1 Task 12 is superseded by the manager-link security and arbitration tests.
- v1 Tasks 14-15 remain required by v2 documentation and final acceptance.
- Completed v1 protocol, runtime, USB, conversion, and panel work is reused;
  it is not deleted merely because the architecture evolved.

## Risks and Stop Rules

| Risk | Stop rule / mitigation |
|---|---|
| HTML implementation exceeds RP2350 resources | Stop after the compile/resource spike and reduce the documented profile before adding features |
| TinyGo/Wi-Fi cannot enforce WPA3-only | Do not ship Wi-Fi; keep USB functional and document the blocker |
| Panel sequence regresses | Stop at golden-frame output; do not experiment on hardware until the known-good sequence passes |
| Public manager weakens device-key isolation | Block the API/link phase until threat-model and negative tests pass |
| A task cannot finish inside its stated file scope | Split the task before coding rather than growing files/functions past limits |

## Preserved v1 Plan

The following incomplete plan is retained verbatim as historical implementation
and acceptance context.

# Implementation Plan: Remote e-paper v1

## Overview

Build the approved modules in dependency order: deterministic panel and frame
foundations, USB-first runtime, host client, authenticated Wi-Fi, then terminal
rendering and measured power/performance polish. Each slice stays buildable and
tested; no commit is created without an explicit request.

## Architecture Decisions

- One 48,000-byte device frame; no hidden second frame.
- One frame protocol over ordered USB CDC and TCP streams.
- USB preempts only incomplete Wi-Fi reception, never a physical refresh.
- Wi-Fi is disabled until authentication, replay protection, and integrity are
  specified and verified.
- Image conversion and terminal rendering run on the host, not the Pico.
- Full refresh is the only stable panel operation; partial refresh remains
  experimental.

## Task List

### Phase 1: Foundations

1. Implement and test the frame record codec.
2. Implement and test frame transaction ownership and CRC validation.
3. Add TinyGo compile, allocation, stack, RAM, and flash resource probes.
4. Implement and host-test the GDEY075T7 panel driver.
5. Build and physically verify tagged panel patterns.

### Checkpoint: Foundations

- Host tests and `go vet` pass.
- TinyGo `pico2-w` builds and resource gates pass.
- Physical A/B/checker patterns are visible on the exact panel.

### Phase 2: USB Slice

6. Specify and implement the device runtime state machine.
7. Implement USB CDC transport and USB-priority arbitration.
8. Implement a Go host client that sends an already packed 1-bit frame.
9. Add host image conversion, scaling, rotation, thresholding, and dithering.

### Checkpoint: USB End-to-end

- Twenty consecutive USB transfers pass protocol and display checks.
- Disconnect, corrupt frame, timeout, reset, and retry paths are verified.

### Phase 3: Wi-Fi

10. Specify and threat-model authenticated Wi-Fi transport.
11. Implement CYW43439/lneto TCP transport with bounded buffers and one client.
12. Verify USB preemption, authentication, replay rejection, and radio budget.

### Phase 4: Linux and Terminal

13. Add text/stdin and terminal-snapshot rendering to the host client.
14. Document macOS, Linux, Raspberry Pi, USB, Wi-Fi, power, and recovery flows.
15. Run final code, simplicity, security, and hardware acceptance reviews.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Similar 7.5-inch drivers use incompatible commands | High | Exact GDEY075T7 source is authoritative |
| CDC input overflow silently drops bytes | High | 256-byte chunks and stop-and-wait ACK |
| Shared frame overwritten during refresh | High | Explicit complete/leased ownership |
| Unauthenticated LAN upload changes display | High | Wi-Fi stays disabled until secure session design |
| Wi-Fi stack consumes too much RAM/energy | High | Fixed buffers, one connection, measured gates |
| E-paper wear/ghosting from frequent updates | High | Coalesce and enforce at least 180 seconds |

## Open Questions

- Physical panel acceptance depends on the connected hardware.
- Wi-Fi credential provisioning and authentication mechanism will be resolved
  in the Wi-Fi spec before that transport is enabled.
