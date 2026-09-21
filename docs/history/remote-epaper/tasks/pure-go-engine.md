# Pure-Go engine migration

2026-09-06. Active task list for the same remote-display initiative. User asks
for autonomous implementation until the complete candidate firmware is ready,
then manual acceptance. No push or additional commits are inferred from the
request to commit the pre-migration work. All commands run in the existing
Dev Container at `/workspaces/pico-sandbox/experiments/03-remote-epaper`.

## Contract and dependency order

ADR-013 supplies module ownership and supersedes Blitz/stylesheet assumptions.
Preserve security, USB priority, bounded streaming, panel sequence and quality
rules. Existing recoverable checkpoint: `37a1471`.

Each numbered task is one testable slice, normally 1–5 files. Split a slice
further before exceeding that scope. Unit tests use `go test ./<package>`;
checkpoints use `./scripts/quality.sh task`, then final `full`. Dependency
qualification also requires real CPU pixel output, `CGO_ENABLED=0` and
`GOOS=darwin GOARCH=arm64 CGO_ENABLED=0` host-tool builds. Firmware builds use
`tinygo build -target=pico2-w -scheduler=tasks -size=short` and resource probes.

## A. Preserve and qualify

- [x] A1. Commit existing accumulated work with expanded evidence and gaps.
  Verify: fast/task PASS, staged whitespace/secret-pattern checks, `37a1471`.
- [x] A2. Map components and record architecture correction in ADR-013.
  Verify: compare latest user decisions against current code and contracts.
- [x] A3. Pin parsers/Canvas; prove native Go text/image output and cross-build.
  Files: go.mod/go.sum and focused renderer qualification tests.
  Accept: no CgoFiles in runtime closure, deterministic bounded raster dimensions.
  Progress: native Go exact-size surface/Cyrillic text and Darwin arm64 build PASS;
  alpha image sampling, object-fit and background tiling now pass pixel oracles.
  License boundary in engine-dependencies.md; final tool cross-build still E4.
- [x] A4. Freeze engine profile and numerical input/paint budgets.
  Files: engine spec, active requirement/map/profile pointers.
  Accept: every property has values/defaults/inheritance/errors and a test plan.
  Evidence: SPEC-engine.md; six independent spec findings reconciled (units,
  operation/byte bounds, opacity=1, defaults/backdrop, transitive LGPL, build tags).

Checkpoint A: dependency acceptance, source/license review, plan review; no
hardware change. Do not build the entire layout on an unqualified dependency.

## B. Input, layout and painting (A3–A4)

- [x] B1. Shared display/viewport profile and checked coordinate/unit resolution.
  Accept: varied/non-byte-width/rotated dimensions, overflow and allocation guards.
- [x] B2. One HTML5 tree, allowlisted tags/attributes, source-free diagnostics.
  Accept: malformed/active/oversize/deep/duplicate input rejects before assets.
- [x] B3. Inline CSS grammar and ordered typed declaration application.
  Accept: no string-splitting parser; shorthand/reset/duplicate/important semantics.
- [x] B4. Box, spacing, dimension and positioning property families.
  Accept: px/%/em/rem/viewport units, finite ranges and valid auto semantics.
- [x] B5. Text, background, image and compositing property families.
  Accept: complete supported value table, unsupported properties/values reject.
- [x] B6. Block sizing, margins/padding/borders and min/max constraints.
  Accept: exact geometry at 800×480 and smaller/larger viewports.
- [x] B7. Font shaping and mixed inline/inline-block wrapping/alignment.
  Accept: Ukrainian, metrics, whitespace, long text, clipping/ellipsis diagnostics.
- [x] B8. Relative/absolute placement and containing-block regressions.
  Accept: static wrappers, right/bottom, percentages and out-of-flow siblings.
- [x] B9. Bounded assets, intrinsic ratio, fit/position and backgrounds.
  Accept: PNG/JPEG contract, corrupt/oversize input, transparent/image regressions.
- [x] B10. Paint order, nested stacking, clips and group-opacity composition.
  Accept: independent RGBA expectations, not just monochrome masks.
- [x] B11. Final monochrome conversion and protected timestamp region.
  Accept: deterministic display-anchored dither, overflow/legibility, no outside writes.
- [x] B12. Engine orchestration, cancellation/work limits and golden previews.
  Accept: bounded complete HTML→frame, stable diagnostics, multi-viewport fixtures.

Checkpoint B: active Go reproductions replace all three skipped Blitz defects;
fuzz invalid input, race shared fonts/state, benchmark bytes/allocations/latency.
Every behavior uses RED→GREEN tests; no unsupported feature silently paints wrong.

## C. Manager and authoring tools (B12)

- [x] C1. Bind native engine into manager and a pure-Go HTML preview/USB tool.
  Accept: no worker executable required; same HTML produces the same target.
  Evidence: manager HTTPS/native engine/real EPS1 codec/two-plane recording sink
  at 17×9; native epaperstream CLI→two-plane sink at 8×1. No hardware claim.
- [x] C2. Atomic by-ID edits with base revision and optional batching.
  Accept: 5 related edits one scene; stale/overlap/delete cases all-or-nothing.
  Evidence: PATCH /v2/screen uses the same strong epoch ETag, authenticated/rate-
  limited 32 KiB admission and existing optional batching queue. At most 64
  disjoint original-ID edits; validated attributes, context-aware fragments,
  canonical structural roundtrip, strict JSON/Unicode and whole-scene assets
  precede CAS. Two implementation-review findings fixed: validator required
  (501 otherwise), root attribute edits allowed while structural edits reject.
- [x] C3. Confirmed-pixel baseline and no-op delivery suppression.
  Accept: identical output avoids transfer/refresh unless maintenance is due;
  failed/unknown ACK invalidates. Test before/at deadline, continuous edits,
  offline recovery and no catch-up refresh burst.
  Progress: exact final-pixel comparison, independently retained confirmed
  bytes, logical Confirmed versus last actual Delivered, unknown-ACK/session
  invalidation and borrowed-storage tests pass. Subsequent screen maintenance,
  cycle-clock and recovery lifecycle/schedule tests close deadline integration.
- [x] C4. Full-cycle timestamp, timezone binding and maintenance scheduling (D1–D2).
  Accept: fake-clock tests, configured 600s default, no failed timestamp advance.
- [x] C5. Harden HTTP input/output and scene/asset retention.
  Accept: auth/body/rate limits, no source/secret leakage, no unapproved fetches.

Checkpoint C: local HTTPS→real Go render→fake USB sink integration, stale render
completion and API recovery. Compare emitted frame with the inspected preview.

## D. Unified firmware and transport (C1, existing secure modules)

- [x] D1. Freeze versioned screen session, target identity and diagnostics.
  Accept: complete record lengths, caps/profile, status/reconcile and error stages.
  Evidence: docs/eps2-wire.md, screenwire strict codec/capabilities/status,
  paneldiag numeric phase/step/command/BUSY mapping, independent malformed-byte
  tests. Review reproduced/fixed impossible terminal progress and exact reply
  correlation. Runtime composition/physical qualification remain D3–D9.
- [x] D2. Preserve receiver completion across transport reconnect safely.
  Accept: lost ACK does not blindly refresh again; reboot/unknown forces resync.
  Evidence: streamsession → screenlink → screenclient retain boot-scoped lease,
  private random acquisition claim, connection serial, consumed-ID fence and
  one terminal result. Lost Acquire/Commit ACK, competing owners, changed boot,
  stale disconnect, late Commit failure and abort-evidence tests pass. No pixels
  retained by session/client; new receiver/SHA state is bounded, not allocation-free.
- [x] D3. USB streaming plus physical provisioning/control dispatch.
  Accept: existing atomic store/rotation/factory reset; blank device USB functional.
  Rotation/erase invalidates old sessions and incomplete staging, rejects old
  keys and reloads complete config; invalid/erased config disables networking.
  An already-started physical refresh finishes safely before reconfiguration.
- [x] D4. Bind authenticated encrypted records to screen streaming.
  Accept: authenticate before SPI; replay/tamper/truncation reject, complete-before-visible.
  Progress: actual securetransport records → EPS2 → recording physical driver
  integration passes, including production bridge/peer/owner composition.
- [x] D5. Single device owner and USB preemption across transports.
  Accept: incomplete WiFi aborts; active physical refresh cannot be interrupted.
- [x] D6. Compose WPA3-only Pico-initiated IPv4 connection and reconnect policy.
  Accept: no WPA2 downgrade/Bluetooth, waits for server; public address not filtered.
- [x] D7. Bind rendered manager targets to the authenticated device connection.
  Accept: USB/network share payload, enrollment/key isolation and bounded queues.
- [x] D8. Negotiate compact encodings only where replay proves equivalence.
  Accept: raw baseline, decoded-work bounds, actual total wire-byte measurement.
  Evidence: docs/eps2-compression.md; negotiated PackBits uses one 1024-byte
  decode scratch, exact-length checks, malformed-block transaction abort and
  canonical two-plane/SHA validation. Whole encrypted HTML exchange 114072 →
  22334 bytes with identical recorded physical driver trace. Target RAM/energy
  and optical acceptance remain E4/E5, not inferred from these counters.
- [x] D9. Compose generic secret-free Pico 2 W UF2 and diagnostics.
  Accept: shared firmware USB/WiFi/provisioning, exact target, RAM/flash/stack report.

Checkpoint D: realistic fake panel/transport fault matrix, malformed/reordered
records, resets, timeout/power cleanup, no unexpected full-frame Pico storage.
No hardware pins, plane polarity or accepted timing changed without primary sources.

## E. Removal and final evidence (B–D)

- [x] E1. Remove Rust worker and obsolete Go IPC consumers after replacement.
  Accept: no active Blitz imports/processes/config; port useful tests first.
  Exact scope: blitz-probe source/manifests, blitzworker, cmd/blitz-preview and
  their build integration. Preserve historical ADRs and evidence in Git/docs.
  Evidence: docs/blitz-removal-map.md and exact 67-file manifest; all files were
  tracked/unchanged at 37a1471, then removed after native assertion ports passed.
  Ignored worker/PNG/build artifacts, fixtures and historical reports retained.
- [x] E2. Remove Rust/Stylo-only Dev Container dependencies and obsolete commands.
  Accept: reproducible Go tooling config; no rebuild/container replacement by agent.
  Evidence: Dockerfile removes only the Rust checkpoint additions; Pico SDK/C/
  Python prerequisites remain for preserved physical experiments. Old entry
  commands/Stylo exceptions are marked historical. No rebuild was performed.
- [x] E3. Review/fix/simplify per module and full integration.
  Accept: lint/coverage/race/fuzz, CGO=0 server/tool builds, TinyGo artifact all pass.
  Complete oracle: real HTTPS submission → native renderer → actual USB/encrypted
  codec → unified runtime → recording panel; both streamed planes must equal
  the preview target. Include preemption, bad auth and lost completion.
  Evidence: 2026-09-07 full gate PASS in 82s; 91.9% changed and 92.1% total.
  Late engine/coverage-collector independent reviews have no required findings;
  package/archive script review remains part of E4, not physical acceptance.
- [x] E4. Package candidate firmware, macOS tools, fixtures and operator guide.
  Accept: checksums, reproducible build instructions, explicit known limitations.
  Evidence: `build/native-candidate.gEnGoa`, full gate PASS in 74s, all 130
  manifest entries verified, Linux/Darwin vulnerability audits clean. UF2 target
  RP2350 and exact review-build hash verified; both package PNGs inspected.
  See `docs/native-candidate-acceptance.md`; no flash or hardware claim.
- [ ] E4 follow-up after hardware fixes: align release-script TinyGo pin and
  third-party notices/source checks with accepted0.42.0, then package into a new
  directory. Current script still requires0.41.1; preserve the frozen E4 bundle
  and accepted separate UF2/sender. Do not downgrade to make packaging pass.
- [ ] E5. Manual acceptance handoff only after all automated work is ready.
  Accept: user checks image, repeat update, WiFi and recovery; physical outcomes
  remain unchecked until observed. Partial/energy acceptance is never fabricated.
  - [x] First native HTML USB image: `screen=confirmed`, exit0 and independent
    user confirmation «Так, новий dashboard видно» on 2026-09-07.
  - [ ] Repeat changed HTML without reflashing; compare content and corner time.
    Protocol confirmed on 2026-09-07 by 13:26:38 UTC, same accepted firmware
    and sender; visible acceptance FAILED: user still sees «Мій e-paper
    dashboard», not «Кадр №2». Investigate before more images or WPA3 tests.
  - [ ] Physical reconnect and interrupted-transfer recovery.
  - [ ] WPA3 provisioning, authenticated manager delivery and USB priority.
  - [ ] Separate partial-refresh and measured peak RAM/energy acceptance.

Current E5 status: first native USB image accepted; repeated image failed visible
acceptance. Diagnostic unified TinyGo0.42.0 `screen-device-panel-trace.uf2` is
now installed; one controlled upload returned confirmed. Cached cycle1 trace:
completed,2054ms total, each of three BUSY waits samples1/LOW0. New visual
acceptance is awaiting the user; no physical cause inferred from this alone.
Panel sequence and 180-second floor remain unchanged. Tests/race and quality
task PASS21s (changed96.1%, total91.8%), not proof of physical repair. Exact
candidate and preserved accepted hashes are in the debugging history; never
substitute the older frozen E4 package executables.
Migration and first-acceptance evidence were committed and pushed as `66dc9a1`
after explicit user approval. Other E5 gates remain open; no Wi-Fi enrollment.
The E4/resource numbers below describe their historical build, not this UF2.

## Baseline review findings to close

Independent read-only review, 2026-09-06:

- stream-device is USB-only, no provisioning or arbiter (D3–D6).
- manager screen mode hardwires serial delivery (D7).
- disconnect clears completion identity; client always uses intent 1 (D1–D2).
- identical scenes still cause full refresh (C3).
- panel errors collapse to generic code 7, losing stage/command/BUSY (D1).
- timestamp is isolated, not live scheduling (C4).

Plan review: all four subsequent findings were actionable and incorporated:
live credential invalidation, no-op/maintenance distinction, C4 dependency on
D1–D2, and the complete software-path oracle. No external model CLI was used.

## Stop rules

No quality/security weakening, test skips or unsupported visual promises.
Record unexpected dependency limitations and choose a documented, tested
alternative within the approved direction. If completion truly requires new
authority or hardware evidence, report the exact boundary; do not label a
USB-only prototype or successful compile as the complete product.

## Current implementation evidence

2026-09-07 software closure for B/C/D (does **not** close E5):

- B1–B12: whole engine race PASS (engine 94.0%; DOM 93.9%; geometry 93.2%;
  style 95.2%), strict lint zero. Real multi-viewport previews inspected, all
  three historical ignored defects have active native regressions. Independent
  late review fixed inline baseline/slicing, clipped absolute descendants and
  a pinned Canvas text/object merge issue via our face-identity adapter. The
  explicit non-browser combination boundaries remain in SPEC-engine.md.
- C3–C5: exact confirmed-pixel/no-op tracking, fake-clock maintenance/cycle
  tests, recovery fences and HTTPS admission tests pass. Direct USB now shares
  the timestamp painter; CLI race coverage 93.2%. Native manager USB uses the
  retained EPS2 sender. Loopback/trusted-author restrictions remain; this is
  not approval to expose a listener publicly.
- D3–D7/D9: one compiled generic firmware binds USB provisioning, durable
  epochs, encrypted records, USB priority and WPA3-only networking. Production
  bridge/peer/owner integration compares the complete recording-driver trace
  with the expected image and checks lost ACK/preemption. Host fakes do not
  validate physical radio, flash or CDC. Upstream radio cancellation limits are
  documented, not repaired or hidden. MCU IPv6 and partial refresh remain out
  of this accepted baseline.
- Resource closure: TinyGo 0.41.1 uses Go 1.26.8. Flash 696,380 bytes; linked
  pre-heap SRAM 34,704, plus four task allocations 33,024. Peak heap/stack and
  electrical energy remain unmeasured. See native-firmware-resources.md.
- Latest task gate: PASS in 30 seconds, changed coverage 91.9%, total 92.1%
  versus 75% baseline. Final packaging/full gate still recorded separately in
  E3/E4; no new hardware test, flash, commit or push has been performed.

2026-09-07 migration-gate correction: the coverage wrapper unioned
`base...HEAD` and `HEAD→worktree` hunks. After approved renderer removal this
resurrected deleted intermediate files; inserted lines also left stale
coordinates. Two real temporary-Git reproductions failed before the fix.
Collection now uses one `merge-base(base, HEAD)→final worktree` diff plus
untracked files. Deleted code is absent; staged/unstaged/branch code is measured
at its actual final lines. Missing executable profiles still count uncovered;
the 90% threshold, baseline and no-skip rules are unchanged. Tools race and
strict lint pass. Source: [Git diff](https://git-scm.com/docs/git-diff).

2026-09-07 integration follow-up (software evidence only):

- `screendelivery` is the shared physical-delivery contract. `screenhub` owns
  bounded EPN2 admission (four handshakes/current/queued), while only the manager
  pump owns EPS2 client identity and resolution. Dropped data and Commit replies
  are reconciled without automatic repeat refresh. Same-boot quiet reconnect,
  reboot resync, profile checks and socket cancellation have actual AEAD/EPS2
  tests. Strict lint and race pass; hub coverage 91.6% after review fixes.
- Independent review fixed idle `Close` not waking the pump, and protocol/auth
  failures being misclassified as transient when joined with a close error.
- Manager `--screen-transport wifi` now binds that hub to USB-created enrollment;
  `--maintenance-interval` defaults to 10 minutes. Zone data is embedded only in
  the manager binary, because the minimal container has no system zone database.
  Scoped manager-command race suite passes, coverage 89.8%, lint zero issues.
  USB command migration and public-input gates remain in progress.
- New relative-position tests reproduced percentage-height and RTL precedence
  defects. Fixed without changing following-block flow; whole engine tests and
  lint pass. Sources and exact behavior are recorded in SPEC-engine.md.

The latest whole native `go test ./...` passed before the relative-position and
Health/USB-tool slices. No new UF2 has been flashed. All final E-stage gates and
manual acceptance are still required; this is not a completion claim.

2026-09-06, following checkpoint commit. Not a firmware-ready claim:

- Added manager-only engine/style: grammar-based inline declarations, typed
  lengths, shorthand/important priority, colors, image placement, text styles,
  explicit inheritance and property-specific font/line-height resolution.
- Added engine/dom: one HTML5 tree plus raw lexical guards; no stylesheets,
  active content, URL fetches or duplicate attributes/IDs. x/net v0.58.0 drops
  duplicate attributes before returning tokens; a bounded lexical guard checks
  raw start tags before normalization. Negative regression demonstrated the bug.
- Added independent engine/geometry: bounded content/border boxes, percentage
  bases, min/max, auto margins, inline-block shrink sizing and indefinite heights.
- Extended rasterasset with bounded JPEG decoding; PNG path/tests retained.
- Embedded all eight Go regular/mono weight/style variants. Fonts stay server-side.
- Independent RED tests caught ratio underflow, case handling and computed-error
  provenance; further reproductions cover quoted families, image-axis grammar,
  inline-block auto margins and accidental resource-limit clamping. All fixed.
- `go test ./engine/... ./rasterasset` and focused strict lint PASS. Style race
  checks passed before the latest geometry additions; final whole-project gates,
  fuzz, compositing, renderer/manager/firmware integration remain required.

No existing runtime removed yet; no new firmware flashed and no push performed.

Further incremental evidence (same migration, still not firmware-ready):

- `engine.New().RenderRGBA` now joins DOM → styles → box/text layout → native
  Canvas glyph paths + Go image sampling → RGBA. Font caches are renderer-owned
  and admission is serialized/context-aware. No image bytes or text goes to MCU.
- Exact layout tests cover several viewport widths, non-collapsing margins,
  absolute padding-box insets, relative movement, root percentage height,
  mixed Cyrillic/bold text, line height and atomic inline-block placement.
- RGBA oracles cover source alpha over the actual backdrop, once-only group
  opacity, nested z-index isolation, ancestor clipping and image fit/repeats.
- Independent review found five slice defects: paired absolute auto dimensions,
  synthetic-document percentage height, intrinsic fixed-child widths,
  inline-block paint order and static opacity z-index. All reproduced and fixed.
- Task coverage initially rejected the slice at 85.7%. Added direct geometry,
  presentational-hint, invalid-input, cancellation and resource-limit tests;
  focused coverage is engine 90.1%, geometry 93.0%, style 95.4%, DOM 85.7%.
  Aggregate task recheck and later full gates remain authoritative.
- Still pending: complete whitespace/overflow/inline-edge semantics, warnings,
  protected stamp and mono adapter, manager and unified firmware integration,
  exact historical image-bug ports, removal, fuzz/race/performance and packaging.

Next completed slices (supersede the corresponding pending items above):

- Native PNG/mono/BZM1 `cmd/engine-preview`, complete mono packing, independent
  result ownership, odd-width stride and display-anchored quantization tests.
  Visually inspected `build/engine-dashboard-rgba-v2.png`: readable Ukrainian,
  inline-block cards and intact rounded corner arcs. No physical screen update.
- Absolute spans blockify; relative inline percentage offsets use the nearest
  block container. Inline positive decorations advance content; merged per-line
  fragments prevent duplicate backgrounds. Positioned inline containing bounds
  and list bullet/decimal markers have focused tests.
- Independent slice review: square alpha corners, own rounded image clips,
  background origin, ellipsis line metrics fixed via reproductions. Mixed inline
  white-space modes now explicitly reject with the offending CSS range; supported
  block/inline-block modes remain. Combination boundaries recorded in SPEC-engine.
- Optional protected metadata rectangle with overlap warning, not a bottom
  strip; actual refreshstamp scheduling remains C4/D1–D2 work.
- All three old skipped Blitz defect cases have active native Go regressions.
  IMG-001/POS-001 retain exact masks. IMG-002 retains its 4×4 box/3×3 clipping
  oracle and explicitly tests bilinear 63/191 gray samples before Bayer output;
  expecting the old hard-edge mask was an incorrect cross-renderer assumption.
- `quality.sh task` PASS in 17s: changed coverage 91.3%, total 90.3%. This was
  before the subsequent list, native-manager and USB binding changes; rerun gates.
- `go test -race ./engine/...` PASS; 5s bounded fuzz campaign executed 63,265
  cases with no crash. Initial native benchmark (3 iterations, Linux arm64
  Dev Container): warm 25.87ms / 12,039,960 B / 30,298 allocs; cold 29.27ms /
  18,767,200 B / 53,451 allocs. These are server allocations, not Pico RAM, and
  do not measure electrical energy or establish a worst-case bound.
- Manager `--screen` replaces `--render-worker`; width/height configurable.
  Loopback/trusted-author restriction retained until C5 acceptance. Pure-Go
  `epaperstream [-width N -height N] FILE.html SERIAL` no longer needs a renderer
  executable. Its supervised `--raw` USB subprocess remains for bounded OS I/O.
- Native manager exposes source-free render warnings and copies them across
  status ownership; render calls receive a 15s context deadline.

Still required: remaining engine edge/resource validation and final previews;
C2–C5, D1–D9, E1–E5. Existing Rust sources/config deliberately remain until the
replacement consumers and tests pass. No new migration commit, push or flash.

2026-09-07 continuation (supersedes the stale C2/D1/D2 pending references above):

- Strict DOM patch/CAS, no-op pixel baseline, EPS2 codec/session/link/client and
  bounded numerical panel diagnostics now have focused regression coverage.
- Actual httptest HTTPS → native engine → AEAD EPS2 → recording panel equals
  the accepted buffered driver's entire SPI/GPIO/delay trace. Wrong key and
  tampered Begin produce zero panel I/O. This is software, not visible evidence.
- `quality.sh task` PASS in 19s before the subsequent D3 control additions:
  changed coverage 92.9%, total 91.4%. Focused race integration/session/wire/link/
  client PASS. Legacy USB flash/static RAM 237724/105564; Wi-Fi 691172/107932.
- Maximum 1056-byte plaintext TinyGo resource build adds 8 flash bytes and no
  static RAM to the preceding Wi-Fi baseline. `-print-stacks` reports recursive
  unresolved paths through runtime.nilPanic, not a measured stack upper bound.
  Unified firmware dynamic heap/stack acceptance remains D9.
- D3 in progress: USB record-boundary multiplexing, strict provisioning v2,
  source-free result codes, unknown-storage state, poisoned ambiguous host I/O,
  credential/session fencing and authoritative post-operation reload. Review
  caught optimistic erase success; independent RED reproduction now passes with
  storage error + actual state. No mandatory new physical check requested.

C3 maintenance, C4 stamp, C5, production owner/network composition D3–D9 and
retirement/package E stages remain. Existing Rust stays until replacement gates.

2026-09-07 independent `stacknet` lifecycle increment (D6 software evidence):

- Fact: the adapter now uses existing asynchronous pinned `lneto` APIs with
  caller-owned cancellation/deadline checks. A fresh stack follows each
  quiescent reset; DHCP result extraction/adoption and packet routing share a
  network-only lock. Validated router identity is copied once.
- RED → GREEN: real TCP active-open was rejected before its first SYN egress;
  three failed local dials leaked all ARP query slots; concurrent `Conn.Abort`
  raced with upstream routing's raw connection-ID read. Fixed using locked
  pending-state getters, public ARP cleanup and outer identity synchronization.
- RED → GREEN: DHCP subnet network/broadcast addresses and router/address
  conflicts, plus limited-broadcast manager/DNS addresses, were admitted.
  Focused rejection tests now pass; `/31` host semantics remain covered.
- Measurement: actual in-memory Ethernet/UDP DHCP DORA + gateway ARP, DNS
  success/error/timeout, TCP handshake/payload, cancellation and concurrent pump
  tests pass. Focused race run reports 95.0% statement coverage; strict package
  lint reports zero issues. No skipped tests, suppressions or dependency edits.
- Boundary: one serialized worker closes each borrowed socket before another
  dial/reset; `Configure` requires a fresh reset and does not implement in-place
  renewal. These contracts and exact primary sources are recorded in
  `docs/wifi-runtime-constraints.md`. Whole-runtime integration/build and physical
  WPA3/USB/panel acceptance remain separate parent-task gates; not claimed here.

2026-09-07 manager full-cycle scheduling increment (C3/C4 software evidence):

- Opt-in `NewScreenWithOptions` binds the enrollment timezone, protected-corner
  renderer, trusted wall-clock source, and positive configurable maintenance
  interval (zero defaults to 600s). Legacy constructors stay compatible.
- Canonical unstamped baseline comparison suppresses pixel-equivalent author
  updates without advancing physical delivery history. Every actual full send
  gets a separate cycle ID and copied/stamped frame; exact `ResolveCycle` is
  required, and maintenance never increments the author revision/ETag.
- Due maintenance coalesces the newest pending scene before optional debounce,
  never before device cooldown. Superseded rendering is discarded; a diagnosed
  invalid latest scene can maintain only the older confirmed baseline, retaining
  its older delivered revision and the explicit rejection. No catch-up burst.
- RED → GREEN: completion accepted elapsed time older than a concurrent Submit;
  shared queue clock validation now rejects it. A bare renderer rejection also
  now fails closed instead of allowing undiagnosed maintenance fallback.
- Fake-clock and real-render tests cover before/at deadline, failure/stale ACK,
  separate wall/elapsed clock changes, forced latest/rejected rendering, offline
  no-burst behavior, exact timezone glyph pixels, preserved viewport/warnings,
  and immutable historical proof. The scoped race run passed: manager 95.8%,
  engine 92.8%, refreshstamp 100.0% statements (before later pump-recovery tests).
- Contract and primary time source: `docs/screen-maintenance.md`. A clock error
  after terminal success invalidates timestamp proof; it does not deny that the
  physical cycle received success. Retained authenticated reconnect integration,
  full task gates and final physical acceptance remain parent-owned; no hardware
  action, dependency change, suppression or commit was performed in this slice.

2026-09-07 additive cached network health (D9 software diagnostics):

- Added EPS2 Health=10 without renumbering Reply=9 or changing old reply bodies.
  The zero-identity, empty request returns status[48]+versioned health[8]; strict
  codecs reject unknown operations/version/state and mismatched body lengths.
  `screenclient.ReadHealth` sends exactly one query, no Acquire/Bind or retry.
- Unified composition installs the callback before tasks. State/last-failure/
  count are one atomic worker snapshot; failures retain the pre-Backoff stage,
  exclude expected owner cancellation, and saturate. Monotonic runtime uptime
  also saturates. Optional network construction failure reports setup-failed=7
  through normal USB without probing/restarting Wi-Fi or fabricating a boot ID.
- Real session/sink regression proves Health does not Tick expired staging,
  acquire ownership, or call the panel sink. A following independent owner Tick
  still produces the expected timeout/Abort. Existing USB DTR preemption and
  expiry remain: opening USB may abort another transport's incomplete staging.
- Focused race tests PASS: screenwire 98.7%, screenlink 95.4%, screenclient 94.9%,
  screenwifi 95.4%; strict package lint zero issues. TinyGo Pico 2 W tasks build
  PASS: flash 694484 bytes, static RAM 8512 bytes. This is not a dynamic heap/
  stack bound or hardware acceptance. Full task gate remains parent-owned while
  manager CLI policy integration is in progress.
- Contract: `docs/device-health.md` and the additive section in
  `docs/eps2-wire.md`. No GPIO/power sequence, dependency, firmware wire default,
  skip, suppression, flash or commit was changed/performed for this increment.

2026-09-07 native inline layout correction (C5 software evidence):

- RED → GREEN geometry: visible-overflow inline-blocks previously bottom-aligned
  instead of exporting their last in-flow line baseline, including nested block
  descendants and text extending beyond an explicit height. Empty, replaced and
  non-visible-overflow cases retain bottom-margin alignment. An outside list
  marker overwrite was separately reproduced and fixed.
- RED → GREEN exact RGBA: wrapped inline borders previously repeated left/right
  edges and rounded broken corners. Virtual stitched decoration now slices them
  in LTR/RTL while retaining background continuity and existing raster budgets.
  Unequal-height, aspect-scaled backgrounds and asymmetric RTL padding/borders
  each received their own failing regression before correction.
- Whole engine race suite PASS: engine 93.7%, DOM 93.9%, geometry 93.2%, style
  95.2% statements in the initial completed scoped run. Strict whole-engine lint
  zero issues. New baseline and shared background-position helpers reached 100%
  in that profile. Final parent integration gates remain separate.
- Contract/source references: `docs/engine-inline-layout.md` and `SPEC-engine.md`.
  Existing one-fragment-per-owner-per-line and unsupported-combination boundaries
  remain explicit. No positioned-layout file, dependency, hardware sequence,
  suppression, skipped test, or commit was changed/performed by this slice.

2026-09-07 final native engine integration review (C5 software evidence):

- Fresh RED → GREEN fixtures closed non-replaced inline overflow erasing text,
  atomic objects missing their surrounding inline owner's fragment, and absolute
  subtrees incorrectly clipped by intervening non-containing overflow ancestors.
  The permitted static-position estimate now retains its paragraph origin rather
  than an uninitialized global origin; the estimated, not hypothetical-layout,
  contract is explicit in `SPEC-engine.md` and `docs/engine-inline-layout.md`.
- Visual PNG inspection exposed missing post-break text beside synthetic inline
  objects. Direct pinned Canvas `dae8cd8e19a7` reproduction localized its same-face
  text/object span merge; a small adapter-only face descriptor copy preserves the
  boundary. The upstream reproduction stays active, a distinct-face control
  preserves all text, and exact second-line RGBA rows match plain text. No font,
  shaper, image payload or upstream code is copied/patched. Independent fresh
  ownership/resource review found no required issue.
- Compact multi-viewport fixture tests cover 250×122, 400×300, 800×480 and 480×800,
  with no warnings, exact anchored pixels, stable repetition and independent
  output storage. Final whole-engine race PASS: engine 94.0%, DOM 93.9%, geometry
  93.2%, style 95.2%; strict whole-engine lint zero and diff check clean.
- Warm dashboard benchmark, three iterations: 27.21 ms/op, 12,033,632 B/op,
  30,304 allocs/op. These are a host sample and cumulative allocation counts,
  not a peak-heap bound or physical acceptance. Corrected PNGs were generated
  through `cmd/engine-preview` and visually inspected at 250×122 and 800×480;
  a desktop dashboard also demonstrates intentional author-layout clipping/
  absolute-footer overlap at small viewports, not automatic canvas scaling.
- No new feature family, dependency change, hardware action, skipped test,
  suppression or commit. Engine files are released for the parent integration
  and final task/release gates.
