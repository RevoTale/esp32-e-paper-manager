# Render conflicts and module boundaries

Recorded: 2026-09-05. Decision: compose one coherent target before encoding.
Status: `renderdiff` is implemented and host-tested; Blitz, snapshot scheduling,
command encoding and physical acceptance of this new path remain integration work.
This supplements `render-design-review-2026-09-05.md`, not the old Pico HTML path.

## Facts and decisions

**Fact:** CSS painting order uses stacking contexts, tree order and paint phases,
not a global numeric sort of every node's z-index. A child cannot escape its
parent context merely by having a larger index. Source:
https://www.w3.org/TR/CSS22/zindex.html#painting-order

**Fact:** group opacity applies to the rendered group, not independently to
each child. Applying 50% opacity to two overlapping children separately changes
the overlap compared with applying it once to their group. Source:
https://www.w3.org/TR/css-color-3/#transparency

**Fact:** compositing depends on the backdrop and isolation boundaries. An
independently flattened transparent subtree is not generally reusable over a
different backdrop. Source (Candidate Recommendation Draft, not final REC):
https://www.w3.org/TR/compositing-1/#groupcompositing

**Unknown:** these semantics on our exact locked Blitz build. The live feature
table is not runtime acceptance: https://blitz.is/status/css. Unsupported
semantics must produce a capability diagnostic; bitmap fallback cannot repair
semantics that the same renderer does not implement.

**Update, 2026-09-05:** the five focused RGBA tests in `../blitz-probe` now pass
on locked Blitz 0.3.0-beta.1 with CPU rendering: basic/nested/equal z-index,
group opacity, and backdrop-dependent alpha. This resolves the unknown only
for those fixtures, not the whole CSS profile. See its README for evidence
and the narrowly approved Stylo Python build-generator exception.

**Decision:** all outputs in a transaction derive from one immutable scene and
asset snapshot. The server resolves stacking, clipping, group opacity and image
alpha before monochrome conversion. Pico receives final opaque replacements or
proven-equivalent mask/fill commands, never unresolved transparency or z-index.
This does not require every transfer to be a full bitmap.

## Conflicts and guards

| Conflict | Required guard / acceptance case |
| --- | --- |
| Two edits finish rendering out of order | Capture scene generation before rendering; publish only the current generation. Cancel or discard older work. |
| Two clients edit from an old document | Serialize mutation of canonical scene; require a base scene revision for partial edits. Reject stale writes rather than merging stale HTML snapshots. HTTP `If-Match` is the standard candidate. |
| Independent edits grouped together | Apply edits to canonical scene, then freeze a target. Never concatenate already-rendered patches from different targets. Preserve unrelated edits. |
| Parent deletion and descendant edit | Validate the edit against the current tree; missing targets reject atomically. Node identity must not be accidentally reused. |
| Nested z-index / equal z-index | Preserve renderer paint order and atomic context boundaries; never sort all primitives by a flat integer. |
| Transparent foreground over changed background | Recompose the affected foreground too, even if its HTML/asset hash is unchanged. |
| Transparent custom raster subtree | Flatten with the actual backdrop, or keep RGBA inside manager until final composition. Do not flatten onto white early. |
| Move, remove, shrink, hide, reflow | Damage includes old and new painted extents and exposed backdrop; erasure is an opaque replacement, not an omitted draw. |
| Clip, image fit, text overhang | Use painted bounds after layout, including supported overhang/effects, not just node border boxes. Unproven bounds use full target comparison. |
| Renderer/asset/font completes late | Bind result to scene generation, asset identity and renderer profile; publish a new coherent target, not a loose patch. |
| Byte-aligned dirty bounds expand | Fill expanded pixels from the same final target, never clear them blindly. Clip at right viewport edge. |
| Dither seams / changed phase | Anchor ordered dither to absolute display coordinates. Do not restart error diffusion independently per dirty tile; require full equivalence or a different documented conversion mode. |
| Timestamp corner | Build the protected corner into the final target under the existing full-refresh policy. Partial keeps confirmed corner pixels; expanded bounds must preserve them. |
| Retried/delayed ACK, failed panel refresh | Accept matching epoch/target only; ambiguous state invalidates base and requires resync. No speculative ACK advances the baseline. |
| USB preempts Wi-Fi | One device writer owns the framebuffer. Abort incomplete upload only; do not interrupt physical refresh. Manager resynchronizes after USB changes. |
| Profile/rotation/dither/cache changes | Invalidate cached rendering and base as appropriate; negotiate profile and full resync. Never reuse asset handles across an unknown epoch. |

HTTP conditional-update reference:
https://www.rfc-editor.org/rfc/rfc9110.html#name-if-match
These scheduling/API guards are requirements, not claims of implemented handlers.

## Patch contract and resource behavior

`renderdiff.Plan(base, target, maxRects)` is a manager-side reference planner.
It compares packed logical pixels, ignores stride/padding bits, expands horizontal
bounds to bytes, joins consecutive equal row extents, and returns disjoint
half-open rectangles. Fragmentation beyond the positive caller limit falls back
to one damage bounding box. It neither copies nor retains either frame.
Time is O(packed frame bytes); rectangle storage is O(min(height, maxRects)).
The caller owns immutable confirmed/target snapshots; this API adds no device
framebuffer or manager RGBA canvas. Bandwidth optimality is not claimed.

Encode every region from the same target snapshot. The raw reference uses
replacement semantics, including white pixels; it does not alpha-blend patches.
Go's documented distinction between replacement and source-over is useful for
an independent replay oracle: https://pkg.go.dev/image/draw#Op.
Optimized fill/mask/cache sequences may overlap intentionally, but must preserve
their ordered semantics and produce exactly the reference target. Do not reorder
such operations for batching or deduplicate by rectangle alone.

The existing draft's bounded validate-before-mutation, canonical lengths,
decoded-byte limits, epoch/revision, authentication and single in-flight rules
still apply. Validate the whole transaction before any pixel or cache mutation;
chunked transport is not permission to refresh per chunk. Hardware partial-window
alignment and refresh safety are separate from planner byte alignment.

## Module ownership

| Layer | Responsibility / boundary |
| --- | --- |
| Public manager API (`manager`, future scene coordinator) | User auth, input limits, conditional edits, canonical scene generation, optional batching. No GPIO or device secrets in public responses. |
| Renderer adapter (planned Blitz worker) | HTML/CSS, viewport, stacking, assets, alpha, shaping, deterministic target. No USB, HAT or panel commands. |
| `renderdiff` | Pure final-frame damage plan; depends only on `display` and Go standard types. No network, renderer or concrete panel imports. |
| Screen codec (planned; separate from legacy `protocol`) | Typed bounded operations, negotiation, validation, results. No CSS, scheduling or hardware. |
| Transport (`usbtransport`, `securetransport`, `wifitransport`) | Framing/link security and delivery; same screen payload. No paint semantics. |
| Device runtime (`devicearbiter`, `devruntime`) | Single writer, USB priority, transaction lifecycle, baseline validity. No HTML layout on new path. |
| `display` | Logical size, color, frame and capabilities; reusable across displays. |
| `panel` | Exact controller init/refresh/BUSY/sleep. Never parses API or credentials. |
| HAT electrical adapter (existing wiring, future extraction only if needed) | PWR/DC/RST/CS and signal mapping. HAT revision is not panel/controller revision. |
| MCU binding (`cmd/device` composition root) | TinyGo pins, SPI, clocks, board-specific construction. Do not leak Pico pin numbers into protocol or renderer. |
| `testkit` and package tests | Reusable fakes and pixel/state oracles. Hardware-specific timings remain in panel tests. |

Do not create empty packages/interfaces just to mirror this table. Reuse the
existing modules; extract a narrow interface only when it supports actual
replacement or test isolation. No HAT or MCU wiring is changed by this work.

## Evidence and remaining test gates

**Measurement:** new planner tests first failed because `Plan` did not exist,
then passed with 100% statement coverage. Race tests pass; a five-second fuzz
run completed 589,035 executions. Tests reconstruct the target pixel-by-pixel,
check disjoint/bounded/aligned regions, unchanged inputs, movement and erasure,
fragmentation fallback, non-byte widths, padding and mismatched strides.
These prove the patch planner, not CSS support or wire/device integration.

**Measurement:** `scripts/quality.sh task` passes in 19 seconds: strict lint,
file-length guard, changed coverage 100% (64/64), total coverage 88.3% against
75.0% baseline, and both existing USB/Wi-Fi TinyGo builds. Known untouched
oversized legacy files remain reported debt. `go test ./...` also passes.
The new planner is not linked into those firmware entrypoints yet; their build
success is a regression check, not target acceptance of the new screen path.

Required next gates (not yet executed):

1. Exact Blitz fixtures: nested/negative/equal z-index, group opacity versus
   per-child alpha, transparent image over changed backdrop, clipping, reflow,
   mixed text/image overlap and reserved corner. Compare RGBA before quantization
   as well as final 1bpp; quantization may hide compositing mistakes.
2. Full-render versus optimized-command replay for every fixture at multiple
   viewports, with asset/cache eviction and deterministic dither boundaries.
3. Scheduler model tests: out-of-order render completion, stale edits, atomic
   multi-edit, optional debounce/max_wait, in-flight/pending and disconnect.
4. Codec/device fault tests: truncation at each byte, malformed lengths/counts,
   stale epoch, duplicate conflicting revision, malformed cache reference,
   no mutation on reject, lost ACK and refresh failure/resync.
5. Target memory/latency/energy and visible panel acceptance. No current test
   grants permission to enable experimental partial refresh.
