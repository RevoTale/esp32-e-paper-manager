# Blitz source-removal boundary

2026-09-07. E1/E2 source removal completed after the preservation map was
reviewed. The [67-file manifest](blitz-removal-manifest.txt) names the exact
repository-relative sources/manifests/tests removed with a file-specific patch.
Before deletion every path was verified tracked, regular, not a symlink,
present at checkpoint `37a14717d4192199bf9483455f33e0c4b95e6be8`, and unchanged
against that checkpoint. No recursive deletion, rebuild or deployment occurred.

## Preserved assertions

| Historical source | Active native evidence |
| --- | --- |
| `blitz-probe/tests/images.rs` two ignored image-layout cases | `engine/historical_regression_test.go`: intrinsic 2×2 absolute image and explicit 4×4 contain/clipping geometry; independent specified bilinear/dither pixels |
| `blitz-probe/tests/viewport.rs` ignored static-ancestor case | `TestHistoricalStaticWrapperDoesNotCaptureAbsolute`: exact 24×16 bitmap |
| `blitz-probe/tests/composition.rs` equal-z tree order and repeated complete output | `engine/migration_composition_test.go`: exact overlap pixels and whole RGBA equality at 64×64 and 800×480 |
| `blitz-probe/tests/text.rs` pre-wrap/newlines and clipping | `engine/migration_text_test.go`: explicit-break equality and independent rectangular clipping at 45×12, 90×28 and 90×40 |
| `blitzworker/image_extra_test.go` palette/16-bit alpha conversion | `engine/migration_assets_test.go`: exact straight RGBA normalization through the engine asset path |
| `cmd/blitz-preview/stamp*.go` synthetic stamp and output preservation | `cmd/engine-preview/stamp*_test.go`, `output_failure_test.go`: shared painter, explicit instant, embedded default zone, every output pixel and failure-before-output |

The additional native preservation tests passed without engine production
changes. Preview first failed because `-cycle-start` was undefined, then passed
after the shared native stamp path was implemented. Existing native tests also
cover nested stacking/group opacity, viewport layout, text shaping, HTML/CSS
admission, asset limits, cancellation and source-free diagnostics.

Do not port obsolete BZR/BZE subprocess framing, stylesheet selectors, source
identity manifests or cross-parser bitmap ownership. ADR-013's single HTML5
tree and inline-only grammar replace those boundaries. No claim of identical
Rust/Go font rasterization, interpolation or full CSS conformance is made.

## Targets and exclusions

The manifest contains three Rust build manifests, 19 Rust source files, 27
Rust test files, 14 Go worker files and four old preview files. Active manager,
engine-preview, epaperstream and epaperscreen imports already use the Go engine.
The removed old preview was the final runtime caller of `blitzworker.Worker`.

Preserve `blitz-probe/README.md` as historical evidence and its `.gitignore`.
Preserve the complete ignored `blitz-probe/target/` directory: it includes the
58,763,640-byte debug executable and checked PNG previews. Preserve all other
build artifacts, five `testdata/blitz-*.html` fixtures, historical ADRs/reports,
and old physical C/TinyGo experiments. Do not recursively remove directories.
The checkpoint `37a1471` retains the original source and approved ignored tests.

Keep `frameio`/BZM1, `renderdiag`, `rasterasset`, `refreshstamp` and current Go
dependencies: active native commands still consume them. Mark old execution
commands and `CONSTRAINTS.md` exceptions superseded in append-only meaning,
with links to native evidence; do not silently erase their historical approvals.

## Dev Container and gates

The Dockerfile cleanup removed `RUST_VERSION`, rustup installation,
Cargo PATH/version checks and explicit `curl`/`pkg-config` packages introduced
for Blitz. Keep Python, CMake, Ninja, ARM libraries, picotool and Pico SDK:
they predate Blitz and serve preserved physical experiments. The installed
SDK's RP2350 boot-stage and CYW43 firmware generation explicitly require Python.
No Python execution, host package uninstall or container rebuild is authorized.

`scripts/quality.sh` has no Cargo step. Its firmware renderer deny-lists may
retain `blitzworker` as a forbidden historical dependency. No repository CI
workflow was found. Retain lint, file/function limits, coverage ratchets, race,
TinyGo builds and resource checks. Full native qualification must assert no
worker in active host dependency closures, no CgoFiles with `CGO_ENABLED=0`, and
CGO-disabled native/Darwin-arm64 host builds. Container recreation and physical
acceptance remain separately reported, not inferred from configuration edits.

## Host qualification follow-up

The first six-tool Linux CGO-disabled build passed; Darwin failed specifically
in epaperprovision's Cgo-only detailed serial enumerator. After review, it now
uses the existing epaperctl platform policy: Linux retains actual VID/PID auto
selection; Darwin rejects auto before opening a device and requires an explicit
port. `epaperctl -list` lists names using pure-Go `serial.GetPortsList`, not USB
identity evidence. The legacy `epaperhtml` tool remains historical and is not
part of the candidate host-tool set.

New RED tests also reproduced first-match selection among two matching Picos
and case-sensitive hexadecimal IDs. Both now pass with explicit ambiguity
rejection and case-insensitive identity matching. Original connection/setup
assertions were moved, not deleted, into the non-Darwin test file. Darwin's
auto-rejection test binary cross-compiles; it was not executed on the Linux
container. Provisioning race coverage is 92.8%, with zero scoped lint issues.

Final five operator tools (`epaper-manager`, `engine-preview`, `epaperscreen`,
`epaperprovision`, `epaperctl`) build successfully with CGO disabled for Linux
arm64 and Darwin arm64. Both dependency closures contain no CgoFiles,
`runtime/cgo` or `blitzworker`. The three historical and five added migration
regressions pass under race. Provisioning/hostprovision and retained
frameio/rasterasset/refreshstamp race checks pass; scoped lint reports zero
issues. Whole-tree E3 gates remain coordinated separately while other modules
and packaging artifacts are being changed. No container recreation or physical
Pico/macOS execution is claimed by these compile and Linux-host checks.
