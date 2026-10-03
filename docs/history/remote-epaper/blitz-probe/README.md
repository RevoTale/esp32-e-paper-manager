# Blitz composition probe

Historical checkpoint, superseded 2026-09-07: Rust source/manifests, the Go IPC
adapter and old preview command were removed after native preservation tests.
The commands below describe `37a1471`, not the current checkout. Existing ignored
`target/` binaries/PNGs remain untouched. Use the [native preview](../docs/native-preview.md)
and [removal map](../docs/blitz-removal-map.md). Earlier failures remain evidence;
they are not unresolved limitations or test skips in the current Go engine.

Known engine debt: the static-ancestor/absolute-positioning fixture fails on
pinned beta.2. The user approved skipping only this reproduction, in addition
to the two image exceptions; all original assertions remain. Earlier counts
below are historical checkpoints, not full profile acceptance. See
[`blitz-positioning-gap.md`](../docs/blitz-positioning-gap.md).

Positioning-deferral checkpoint, 2026-09-06: 66 passes, exactly three approved
ignores, zero failures. Formatting and all-target Clippy passed. The explicit
`--ignored --exact` positioning reproduction still fails; no engine fix is claimed.

The subsequent [native declaration core](../docs/native-css-declarations.md)
adds twelve active tests and an opt-in policy. BZR4 policy remains grammar-only;
this is not yet complete native CSS, fallback or public-input acceptance.
The [native selector gate](../docs/native-css-selectors.md) adds eight further
tests. The [target ownership foundation](../docs/css-target-ownership.md) now
maps native selectors to actual DOM/bitmap owners. Local
[declaration binding](../docs/css-style-bindings.md) and checked
[Go/Blitz source identity](../docs/css-source-identity.md) are implemented;
fallback authorization and composited pixel extraction remain open.

Isolated manager-side capability tests, not a production renderer or Pico firmware.
Fixed trusted HTML goes through real Blitz layout and CPU rasterization. Tests
inspect RGBA pixels before monochrome conversion can hide compositing errors.

## Reproducible inputs

- Rust 1.98.1 (`rust-toolchain.toml`).
- Blitz DOM/HTML/paint/traits exactly 0.3.0-beta.2, default features disabled.
- AnyRender 0.13.0 and Vello CPU adapter 0.17.0; no window/GPU backend.
- CSS preflight directly exposes already-locked cssparser 0.37.0, Stylo/traits
  0.20.0 and URL 2.5.8; no additional dependency package or renderer upgrade.
- `Cargo.lock` fixes the transitive graph (209 resolved dependency packages).
- Composition tests supply no fonts, SVG or remote assets. The local worker
  additionally accepts an embedded font supplied by the Go adapter.
  Neither path is a sandbox for arbitrary untrusted HTML.
- Tests use 64x64 and, for the overlap/repeatability case, 800x480 output.

## Measured result: 2026-09-05

Debian Trixie aarch64, existing Dev Container. `cargo test -j 2` compiled in
33.37 seconds after toolchain/dependency installation; five tests passed in
0.05 seconds. This is debug test-suite timing, not production render latency,
peak memory, cold start, energy or network bandwidth measurement.

1. Opaque overlapping boxes respect z-index and repeat identically.
2. A high-z-index descendant stays inside its lower parent stacking context.
3. Equal-z-index boxes follow tree order.
4. Opacity is applied to the composed group, not twice in the child overlap.
5. A translucent foreground is recomposited against white and black backdrops.

Assertions use interior pixels, exact opaque alpha and one-channel-unit tolerance
for 8-bit rounding. They do not establish every CSS stacking/alpha combination.
Clipping, negative z-index, images, glyphs, viewport units, full scene mutation,
monochrome diff equivalence and transport integration remain separate gates.

## Approved build-time Python exception

The original Stylo 0.19.0 build script invokes its Python property generator;
the approved beta.2 graph now uses Stylo 0.20.0 with the same build requirement.
This was discovered **after** the initial Cargo build had already succeeded;
the agent should have checked before starting it under the user's no-Python rule.
No user-authored Python script was created. On 2026-09-05 the user explicitly
approved subsequent execution of this upstream generator inside the project's
Dev Container without repeated approval. Other Python use remains prohibited
without separate permission; do not claim this is a Python-free build.
Python is a build dependency here, not Pico firmware or renderer runtime code.

Run inside the existing container from this directory:

```sh
/root/.cargo/bin/cargo test --locked -j 2
```

The initial Rust/curl/pkg-config installation was temporary and disappeared
after container recreation. Corrected on 2026-09-05: `.devcontainer/Dockerfile`
now includes curl, pkg-config, Rust 1.98.1, rustfmt and clippy, alongside the
existing C build tools and Python prerequisite. Rust's version must match
`rust-toolchain.toml`. No macOS toolchain is installed.

Verified after the user-triggered rebuild on 2026-09-05: Rust/Cargo 1.98.1,
rustfmt, clippy and pkg-config are available. A clean compiled-artifact check
with `cargo test --locked --offline -j 2 --target-dir <fresh temporary directory>`
compiled all dependencies in 27.61 seconds and passed all five tests in 0.05
seconds. Registry sources had already been downloaded; no old target artifacts
were reused. This verifies the probe build prerequisites, not the full future
renderer/firmware integration.

## Local worker slice: 2026-09-05

`../blitzworker` invokes this executable once per scene with a 15-second deadline
and bounded stdout. `../cmd/blitz-preview` writes a PNG without opening USB.
The real Go → Rust → PNG run rendered `../testdata/blitz-local.html` at 800×480;
Ukrainian glyphs, positioning and the black badge were visually checked.

From the parent experiment directory inside the running Dev Container:

```sh
cargo build --locked --manifest-path blitz-probe/Cargo.toml
go run ./cmd/blitz-preview ./blitz-probe/target/debug/epaper-blitz-probe ./testdata/blitz-local.html ./blitz-probe/target/preview.png
```

Local IPC is **not** the future Pico protocol: `BZR1`, little-endian u16 width
and height, u32 font length and HTML length, then font and UTF-8 HTML until EOF.
Reply: `BZM1`, the same dimensions, then row-major MSB-first 1bpp pixels
(1=black, zero padding). Limits: 32 KiB HTML, 256 KiB font, each dimension ≤2048,
at most 1,048,576 pixels. These are provisional local guardrails, not evidence
of safe public HTML execution. Go supplies its existing Go Regular font locally;
font bytes travel only over local process stdin, not USB or Wi-Fi.

RGBA is composited on white before luminance thresholding. Images, dithering,
public-input isolation, persistent-worker energy measurements and the complete
CSS capability gate remain unfinished. Current USB firmware accepts HTML EPU2,
not this bitmap reply: no new-format USB transfer or hardware acceptance yet.
The existing firmware, wiring, panel timings and recovery path are unchanged.

Regression: embedding `bytes.Buffer` exposed its `ReadFrom` fast path, bypassing
the bounded `Write` method during process stdout copying. A named buffer field
removes that interface; the oversized process-output test guards the boundary.

Verification for this slice: eight Rust tests pass; locked all-target Clippy
with warnings denied passes. Focused Go race tests pass (adapter coverage 100%,
preview CLI 89.3%). Repository `scripts/quality.sh task` passes in 36 seconds:
changed-line coverage 96.9%, total 88.4%, USB and Wi-Fi builds successful.
The gate still reports pre-existing file-length debt; it was not suppressed.
Build success is not radio or display acceptance. No new firmware was flashed.

## Image support and known layout limits: 2026-09-06

The user explicitly chose to keep images enabled and skip only the two known
layout reproductions. Full reports and reactivation criteria:
[`BLITZ-IMG-001/002`](../docs/blitz-image-layout-bugs.md). The exact root
`CONSTRAINTS.md` exception is applied; original assertions remain unchanged.
An intrinsic absolute image can collapse to 0×0; an explicit `contain` image
inside a clipped parent can shrink. Neither beta.2 nor PR #605 fixed these cases.

PNG preparation and raw RGBA encoding live in `../blitzworker`. The executable
now uses the shared bounded BZR1/BZR2 decoder and public-resource adapter,
preserving alpha until scene composition. No Rust image codec or network fetcher
was enabled. Root `xmlns` is rejected by Go to preserve validated HTML parsing.

Activation-increment verification: 16 Rust tests pass, exactly 2 ignored; formatting and
all-target Clippy pass. Real-executable tests verify image pixels, rejection of
malformed requests without a frame and BZR1 compatibility. The Go task gate
also passes; this is not complete CSS/public-input or physical acceptance.

Local Go → PNG → BZR2 → Blitz preview, from the parent experiment directory:

```sh
go run ./cmd/blitz-preview ./blitz-probe/target/debug/epaper-blitz-probe ./testdata/blitz-image.html ./blitz-probe/target/preview-image.png
```

The checked preview contains two 128×128 checkerboards and readable Ukrainian
text. Select the supplied `Go` font explicitly as the fixture does; default
font fallback is not established. Exact limits and historical measurements:
`../docs/manager-image-assets.md`. S5 remains the last accepted physical
checkpoint; no new Pico flash/update or dependency patch in this activation.

## Viewport and CSS checkpoint: 2026-09-06

Ten active `tests/viewport.rs` fixtures verify exact packed output for multiple
viewport sizes, nested percentages, absolute/relative positioning, spacing,
min/max width, box sizing, clipping, hidden content and inline alignment.
Full suite now has 26 passes and only the two approved image-layout ignores.
The trusted `../testdata/blitz-viewport.html` preview also renders readable
Ukrainian text through the real Go caller. Scope, commands, sources and remaining
profile gaps: [`blitz-viewport-css.md`](../docs/blitz-viewport-css.md).
No renderer implementation, dependency or physical checkpoint changed.

## Text and cascade checkpoint: 2026-09-06

The following text/cascade checkpoint is complemented by the isolated
[`csscheck` grammar component](../docs/css-declaration-validation.md): thirteen
active tests enforce strict declaration parsing and bounded typed diagnostics.
It is not yet called by the raw renderer or public HTML API. The combined Rust
suite has 52 passes and only the two approved image ignores; no firmware change.

`tests/text.rs` adds nine real-font fixtures using Go Regular from the existing
locked Go module cache. Go and that cached dependency are required; missing
inputs fail instead of skipping. `tests/cascade.rs` adds four exact bitmap
checks. No new runtime dependency or vendored font is introduced.

The trusted `../testdata/blitz-text.html` scene also passes through the real
Go caller to PNG. Scope, commands, open ellipsis/whitespace gaps and the remaining
typed-validation boundary: [`blitz-text-css.md`](../docs/blitz-text-css.md).

## Checked HTML/CSS integration: 2026-09-06

At this historical checkpoint Go used BZR3 for documents with CSS. Styles came from its bounded
canonical HTML tree and share document-level budgets; validation runs before
Blitz constructs a DOM. BZE1 returns a fixed source-free diagnostic, not a frame.
Real subprocess tests prove rejection before font processing and valid image
rendering. The real Go image preview also passes with readable Ukrainian text.

The suite has 66 passes and the same two approved image ignores at this
checkpoint. Raw BZR1/BZR2 capability tests deliberately remain available and are
not public-input validation. Native property/value acceptance and resource
isolation are still open. Exact format and manager recovery semantics:
[`blitz-checked-ipc.md`](../docs/blitz-checked-ipc.md).

## Opt-in CSS source bindings: 2026-09-06

`domscope::SceneScope::styles()` discovers actual-DOM CSS and returns specified
declaration spans with shared native/bitmap targets per rule or inline source.
Fourteen new tests cover UTF-8/escaped spans, source identity, strict policy,
exact cumulative bounds and unchanged same-document cascade pixels. Metadata
collection is off in normal validation; BZR3 and the public-input boundary do
not change. No new engine ignore or dependency.

At this checkpoint checked Go/Blitz source identity remained open; the next
increment below closes that part only. Fallback authorization, computed bounds
and pixel extraction remain open. Contracts and focused test commands:
[`css-style-bindings.md`](../docs/css-style-bindings.md) and
[`css-target-ownership.md`](../docs/css-target-ownership.md).

## Checked source identity: 2026-09-06

Go now emits BZR4 for all scenes, with bounded source ordinals and an explicit
empty manifest when needed. A `ValidatedJob` verifies actual-DOM source identity
before image injection and paints the same document; mismatch returns no frame.
Fourteen added tests bring Rust to 131 active passes and the same three approved
ignores. Actual Go image and parser-repair previews pass. Raw BZR1/2/3 probes
remain compatible, without identity guarantees. Native/fallback policy and
public-input isolation are not activated. See
[`css-source-identity.md`](../docs/css-source-identity.md).

## Sources

- Pinned paint API: https://docs.rs/blitz-paint/0.3.0-beta.1/blitz_paint/fn.paint_scene.html
- Upstream example used for API orientation, not our pinned implementation:
  https://github.com/DioxusLabs/blitz/blob/main/examples/screenshot.rs
- Paint order: https://www.w3.org/TR/CSS22/zindex.html#painting-order
- Group opacity: https://www.w3.org/TR/css-color-3/#transparency
- Compositing: https://www.w3.org/TR/compositing-1/#groupcompositing
- Build generator: https://docs.rs/crate/stylo/0.19.0/source/build.rs
- Rust installation: https://rust-lang.org/tools/install/
