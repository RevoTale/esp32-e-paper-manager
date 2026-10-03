# Manager image input: bounded PNG and Blitz handoff

Historical handoff, superseded 2026-09-07: current Go rendering uses bounded
PNG/JPEG assets without BZR IPC or Rust. Both deferred image geometry cases now
have active native tests. Earlier failures, commands and measurements below
remain checkpoint evidence, not the current support contract. See
[SPEC-engine.md](history/remote-epaper/SPEC-engine.md) and [removal map](blitz-removal-map.md).

Recorded 2026-09-06. Bounded PNG `<img>` support is enabled through Go scene
preparation → BZR2 → the real Blitz beta.2 executable → monochrome bitmap.
Two image-layout bugs remain unresolved. The user explicitly chose to keep
images enabled and skip only their two named reproductions; the root
`CONSTRAINTS.md` exception is applied. Original expected pixels are retained.
Full reports, upstream links and exception expiry are in
[`blitz-image-layout-bugs.md`](blitz-image-layout-bugs.md).

This is a verified local image handoff, not full CSS conformance, public-input
acceptance or a new physical S5 checkpoint. No firmware, USB contract or panel
change was made. Earlier blocked-activation checkpoints below are historical;
the final activation section records the superseding user decision.

## Contract

`rasterasset.DecodeDataURL(src, limits)` accepts the exact prefix
`data:image/png;base64,` followed by standard RFC4648 base64 with required
padding, zero pad bits and no whitespace. This is a deliberately narrow input
profile, not a complete data-URL implementation. Encoded URL parameters,
percent encoding, JPEG, GIF, SVG, filesystem paths and HTTP(S) sources reject.

Limits are positive caller-supplied configuration, never taken from the HTML:

| Limit | Boundary |
| --- | --- |
| `MaxURLBytes` | Entire source string, checked before decoding/allocation |
| `MaxPNGBytes` | PNG source bytes after base64 decoding, not raster size |
| `MaxDimension` | Each width/height from PNG metadata, before pixel decoding |
| `MaxPixels` | Width × height; division avoids multiplication overflow |

The caller receives owned `image.Image` pixels. Native PNG color depth and
alpha remain intact; there is no early white backdrop, threshold, resampling,
metadata API, file read or URL fetch. Only the PNG decoder is called, independent
of other packages registering image formats globally. PNG bytes with a false
MIME label, corruption, truncation or trailing data do not produce an image.
Pixels use their encoded orientation; metadata-driven orientation/color
management and animation are not accepted capabilities of this slice.

Errors are stable sentinels usable with `errors.Is`: `ErrConfiguration`,
`ErrSource`, `ErrLimit`, `ErrImage`. They include no source URL, embedded bytes,
image metadata or upstream error text. A failure returns no partial image.

## Reuse and security boundary

Reuse Go's existing standard `encoding/base64` and `image/png`, not a custom
decoder. The legacy `frameimage.DecodeFile` has a different contract: filesystem
input, global codec selection and a package tied to the older fixed panel frame.
It remains unchanged; the new pure module has no transport/panel imports.

Threats addressed here: arbitrary source schemes, misleading MIME, malformed
base64/PNG, tiny compressed input claiming huge dimensions, excessive source
bytes and loss of alpha before scene composition. Error output does not leak
input. DecodeConfig and Decode operate on the same in-memory source, avoiding
a file replacement between inspection and decoding.

These are **per-image** limits, not a process-memory sandbox or total document
budget. Pixel storage may use up to eight bytes per pixel for 16-bit RGBA,
plus decoder/row buffers and the bounded encoded source. Base64 allocation can
include up to two padding bytes beyond the exact decoded source length.
Do not claim measured RSS, CPU, energy, firmware RAM or bandwidth savings.

Acceptance checklist for arbitrary/public HTML (items 1–2 now implemented in
the bounded scene/IPC path below; broader acceptance remains open):

1. Bound asset count, total source bytes, total decoded pixels and active work
   for the entire frozen scene; enforce those bounds before decoding each asset.
2. Bind assets to an immutable scene and deliver them through a bounded worker
   contract. Do not add unrestricted Blitz networking or filesystem access.
3. Test real Blitz intrinsic size, `object-fit`/`object-position`, clipping,
   alpha over changing backgrounds and final 1bpp output. Unsupported properties
   need diagnostics; a successful decoder is not a CSS capability check.
4. Add remaining codec/profile fixtures (including interlaced PNG), measure
   process resources, and enforce the existing public-input isolation gate.
5. Test the complete host → USB → physical panel image path separately.

## Verification and regression record

Go 1.26.2, Linux arm64 in the existing Dev Container:

- Initial tests failed because the decoder API did not exist.
- An allocation regression test reproduced excess `=` padding bypassing the
  preliminary source-size calculation. Rejecting more than two pad characters
  fixed it; malformed padding and oversized URLs now reject with zero allocations
  in the targeted `AllocsPerRun` checks.
- Unit/race tests pass with 100% statement coverage. Fixtures cover exact byte
  limits and all three base64 padding lengths, unsupported sources, nonzero pad
  bits, each truncated prefix, CRC damage, trailing/concatenated files, huge
  dimensions, palette transparency and 16-bit color/alpha preservation.
- Five-second fuzz targets passed: 109,336 data-URL executions and 125,044 raw
  PNG mutations wrapped in data URLs. These counts describe bounded runs, not
  exhaustive security coverage.
- Independent read-only review found no Required issue. It suggested raw-byte
  fuzzing (added) and an interlaced fixture (still a documented profile gate).
- `sh scripts/quality.sh task`: PASS, 17s; zero changed-file lint issues;
  changed coverage 92.6%, total 89.8% against the 75.0% ratchet. Existing USB
  and Wi-Fi TinyGo builds pass as regression checks, not new target acceptance.

Commands from this experiment directory, inside the running Dev Container:

```sh
go test ./rasterasset -race -cover
go test ./rasterasset -run '^$' -fuzz '^FuzzDecodeDataURL$' -fuzztime=5s -parallel=2
go test ./rasterasset -run '^$' -fuzz '^FuzzPNGBytes$' -fuzztime=5s -parallel=2
sh scripts/quality.sh task
```

## Integration research — parser approved, renderer gate open

Recorded 2026-09-06. Initial research preceded the user's parser approval.

- **Fact:** pinned `blitz-dom 0.3.0-beta.1` exposes
  `BaseDocument::load_resource(ResourceLoadResponse)`. Its
  `Resource::Image(ImageType, width, height, Arc<Vec<u8>>)` accepts decoded
  raster bytes. The document applies the image to waiting nodes by resolved
  URL and invalidates image layout. See the versioned
  [resource API](https://docs.rs/blitz-dom/0.3.0-beta.1/blitz_dom/struct.BaseDocument.html#method.load_resource).
- **Inference / proposed integration:** parse HTML in Go, validate embedded
  PNG assets with `rasterasset` under whole-scene budgets, replace sources with
  scene-local asset IDs, and extend the local worker contract with bounded
  RGBA assets. Supply those assets through Blitz's resource API before final
  layout/paint. No network fetch, second PNG decoder or Pico change is needed
  for this proposed path. Load ordering, pixel format and composition still
  require real-worker tests; source inspection is not runtime acceptance.
- **Approved and added:** use the maintained
  [`golang.org/x/net/html`](https://pkg.go.dev/golang.org/x/net/html) HTML5
  parser instead of custom tag extraction. After explicit approval, pinned
  `golang.org/x/net v0.58.0` in the main module; its Go 1.25 minimum is compatible
  with the project's Go 1.26 toolchain. `go mod verify` passes. That increment
  left other direct versions and the Rust lockfile unchanged. This host-only
  parser is not imported by the TinyGo firmware. The separately approved
  renderer dependency evaluation is recorded below; neither is a full audit.
- **Guard:** parse and reserialize the same normalized document used for asset
  validation, as the parser's security notes recommend. Parsing is not
  sanitization: source restrictions, aggregate limits and the public-input
  isolation gate remain mandatory. Existing firmware and S5 remain unchanged.

## Implemented scene and local IPC boundary

`blitzworker` parses and reserializes one UTF-8 document with scripting disabled,
matching pinned `blitz-html`. It rejects foreign namespaces, `picture`, `srcset`,
missing/duplicate image sources, non-HTML5 doctypes and root `xmlns`. These prevent
Blitz's XHTML auto-detection from switching parser modes after Go validation.
Tests reproduced the `noscript` and doctype mismatches before the guards.
This is an image-source boundary, **not** complete HTML/CSS sanitization.

| Boundary | Enforced limit |
| --- | --- |
| Input and normalized HTML | 32,768 bytes each |
| Parsed tree | Depth ≤64; ≤4,096 nodes |
| Image references | ≤16, including duplicates |
| Unique source strings | ≤32,768 bytes in total |
| PNG source | ≤24,576 bytes per unique image; also bounded by the scene input |
| Raster | Each side ≤2,048; ≤1,048,576 unique decoded pixels in total |

Identical source strings decode once per scene. Palette/16-bit PNGs convert to
tightly packed straight RGBA8; alpha is not flattened onto white in Go. Sources
become scene-local `epaper-asset:<index>` IDs. Partial prepared documents are
never returned on failure; limit exhaustion remains `rasterasset.ErrLimit`,
wrapped with `blitzworker.ErrInput`, not a misleading configuration error.

The prototype `BZR2` extends local Go → Rust IPC, **not USB/Wi-Fi**:

```text
BZR2 | u16 viewport_width | u16 viewport_height
     | u32 font_bytes | u32 html_bytes | u32 unique_asset_count
     | font | normalized_html
     | repeated { u16 width | u16 height | RGBA8[width * height * 4] }
     | EOF
```

Integers are little-endian. Asset count is 1–16; font/viewport/HTML limits remain
the BZR1 limits. The Rust decoder checks sizes and cumulative pixels before
allocation, rejects truncation/trailing bytes, and transfers immutable pixel
storage to Blitz through `Resource::Image`. Go uses `io.MultiReader` instead of
assembling another full RGBA request. Each process still owns its own pixels;
this is neither cross-process zero-copy nor measured RAM/energy savings.

Text-only requests remain BZR1. `src/main.rs` now calls the shared `read_request`
decoder, accepting validated BZR2 assets before rendering. Malformed requests
exit with a typed diagnostic and no bitmap. Real-executable tests and a Go →
binary preview verify image handoff; the two layout exceptions below still apply.

## Historical blocking experiment: image layout in Blitz 0.3.0-beta.1

The synthetic tests run entirely in the existing container, without Pico:

| Fixture | Expected | Observed |
| --- | --- | --- |
| Absolute 2×2 image at (2,1), intrinsic size | 2×2 black square | Loaded raster 2×2, layout 0×0, blank output |
| 4×4 image box, `contain`, clipped by 3×3 parent | Centered contents, box clipped | Box shrinks to 3×2; wrong final pixels |
| Half-transparent white pixel over black/white | RGB 128/255 | Pass, ±1 rounding tolerance |

`load_resource` installs the raster, clears layout cache and marks damage.
Loading before resolve or between two resolves did not fix either failure.
Independent review reproduced both failures and found no missing public image
loading call. Do not compensate by changing expected pixels or forcing CSS
dimensions: callers' layout semantics are the acceptance requirement.

**Source-backed hypothesis:** pinned `blitz-dom/src/layout/replaced.rs` uses
available space as a maximum-size fallback, then clamps using the image ratio.
This may explain the 4×4 → 3×2 shrink. The absolute 0×0 case is consistent with the
same clamp at zero available height; its exact measure inputs remain unobserved.
CSS `object-fit` sizes contents inside the established box, not that box itself.

Upstream [PR #605](https://github.com/DioxusLabs/blitz/pull/605), commit
`086441c2cca0be6ee13df0c7ed23d077cdc515d6`, preserves parent-resolved dimensions.
Read-only inspection of published `blitz-dom 0.3.0-beta.2` found that correction.
**Initial candidate, superseded by the measurement below:** user approval to
evaluate beta.2 was subsequently granted. Both pixel assertions were retained;
no registry/vendor source was edited.

Verification for this increment:

- Go adapter race tests: 100% statement coverage; a five-second markup fuzz run
  passed 237,555 executions. Exact/aggregate limits, palette/16-bit alpha,
  normalization, ambiguous sources and BZR1/BZR2 encoding are covered.
- Rust full suite: 13 pass, **2 fail** in image layout; the failures remain live.
  BZR2 metadata/truncation/trailing-input tests pass; no image assertion is skipped.
- Go task gate passes in 83s: changed coverage 93.1%, total 90.0% against 75.0%;
  zero lint issues, existing file-length debt still reported. USB/Wi-Fi TinyGo
  builds pass without flashing. Locked all-target Rust Clippy passes with
  warnings denied. The Go gate does not include or override the Rust failures.
- Full public-input isolation, CSS background assets, interlaced PNG, resource
  measurements, end-to-end BZR2 and physical image acceptance remain open.

Reproduce from the experiment directory in the existing Dev Container:

```sh
go test ./blitzworker -race -cover
go test ./blitzworker -run '^$' -fuzz '^FuzzPrepareMarkup$' -fuzztime=5s -parallel=2
/root/.cargo/bin/cargo test --locked --offline --manifest-path blitz-probe/Cargo.toml -j 2 --no-fail-fast
```

Layout/parser references:

- [Pinned HTML sink and parser-mode selection](https://docs.rs/crate/blitz-html/0.3.0-beta.1/source/src/html_sink.rs)
- [Pinned replaced-element sizing](https://docs.rs/crate/blitz-dom/0.3.0-beta.1/source/src/layout/replaced.rs)
- [CSS object-fit contract](https://www.w3.org/TR/css-images-3/#the-object-fit)
- [Published beta.2 source archive](https://static.crates.io/crates/blitz-dom/blitz-dom-0.3.0-beta.2.crate)

## Approved beta.2 evaluation — unresolved, 2026-09-06

The user approved the related renderer dependency update. Exact current pins:

| Component | Previous | Evaluated |
| --- | --- | --- |
| Blitz DOM/HTML/paint/traits | 0.3.0-beta.1 | 0.3.0-beta.2 |
| AnyRender / CPU adapter | 0.11.0 / 0.14.0 | 0.13.0 / 0.17.0 |
| Stylo / Taffy | 0.19.0 / 0.12.2 | 0.20.0 / 0.14.0 |
| Parley / Vello CPU | 0.10.0 / 0.0.9 | 0.11.1 / 0.1.0 |

Rust remains 1.98.1; Blitz requires ≥1.89.0 and uses MIT OR Apache-2.0.
Cargo performed a targeted related-group update: 26 package versions changed,
`grid` was removed; 209 registry dependencies remain (210 including this crate).
The original manifest/lock snapshot is retained in the current container at
`/tmp/blitz-beta2-baseline.19JZMH` for comparison, not as a durable recovery file.
No broad unrelated update or new runtime service was introduced.

Read-only review confirms image codecs, SVG, network clients and default Blitz
features remain disabled. The default `DummyNetProvider` does no fetching;
beta.2 also reports it as a no-op. Raw straight RGBA8 resource loading remains
compatible. Beta.2 enables incremental layout and changes style-threading
defaults, so source inspection does not establish identical behavior or energy.

**Measurement:** beta.2 builds, but both image pixel tests fail identically to
beta.1. Full Rust suite: 13 pass / 2 fail; all-target locked Clippy and format
checks pass. Go task gate passes in 49s, changed coverage 93.1%, total 90.0%;
adapter race coverage remains 100%. USB/Wi-Fi builds pass, with no flashing.

Temporary tracing in our adapter, not dependency sources, confirms valid
`width:4px;height:4px`, no CSS maximum, `item_is_replaced=true`, but unrounded
layout 3×1.5. The absolute image has location (2,1) and size 0×0. Final cache
entries mark both known dimensions definite. Trace code was removed; no private
documents were rendered or logged.

**Inference:** PR605 preserves parent-resolved dimensions, but those values can
already be wrong. The separate available-space clamp on the unresolved-size
path remains suspect. Do not assert that removing it fixes all CSS cases until
a narrowly scoped patch and regression suite prove that.

**Additional migration guard:** beta.2 detects XHTML from root `xmlns`, even
without a doctype. The new Go regression first failed; rejecting root namespace
declarations now retains HTML parsing. This does not turn the adapter into a
complete sanitizer or public-input sandbox.

At this checkpoint BZR2 activation stayed blocked. A versioned upstream
correction or approved narrow dependency patch was proposed, not applied.
The later user clarification below supersedes that activation gate; authored
CSS, original pixel assertions and BZR1 compatibility remain unchanged.

The read-only review found the remaining clamp in upstream file revision
[`df9d7c3d8572c43a74963d07d62364418784f52e`](https://github.com/DioxusLabs/blitz/blob/df9d7c3d8572c43a74963d07d62364418784f52e/packages/blitz-dom/src/layout/replaced.rs#L219).
Proposed experiment: remove only `.or(available_space.into_options())` and
`.maybe_min(available_space.into_options())` from `max_size` resolution. This is
not applied or verified. Add zero/smaller available-space and explicit CSS
min/max tests; preserve both failing pixel fixtures. A local dependency copy
needs an explicit bounded vendor policy and approval, not a registry edit.

The legacy real Go → BZR1 → beta.2 preview also ran successfully. Its
`blitz-probe/target/preview-beta2.png` contains readable Ukrainian text and
positioned boxes; this is a local smoke check, not exact old/new pixel
equivalence or a new display update.

Versioned implementation references:

- [Beta.2 replaced layout](https://docs.rs/crate/blitz-dom/0.3.0-beta.2/source/src/layout/replaced.rs)
- [Beta.2 HTML parser-mode detection](https://docs.rs/crate/blitz-html/0.3.0-beta.2/source/src/html_sink.rs)
- [AnyRender CPU adapter manifest](https://docs.rs/crate/anyrender_vello_cpu/0.17.0/source/Cargo.toml)

## User-approved activation with known layout debt — 2026-09-06

The user clarified that only the two tests should be skipped, not image support.
`main.rs` now reuses the existing bounded decoder; no second codec or network
fetcher was added. Only `images_use_intrinsic_size_and_position` and
`image_contain_and_clipping_use_final_layout` are ignored with bug IDs.
Expiry/ownership and full reproduction details live in the root exception and
`blitz-image-layout-bugs.md`; all other tests stay active.

Verification: 16 Rust tests pass, 2 ignored; format and locked all-target Clippy
pass. Go task gate passes in 62s (changed coverage 93.1%, total 90.0%, zero lint
issues). Real executable tests cover image pixels, malformed input without a
frame and BZR1 compatibility. Review strengthened the aggregate-limit fixture
to contain complete pixel data, so EOF cannot mask a relaxed limit.

Local smoke command, from the experiment directory in the existing container:

```sh
go run ./cmd/blitz-preview ./blitz-probe/target/debug/epaper-blitz-probe ./testdata/blitz-image.html ./blitz-probe/target/preview-image.png
```

Observed output: two 128×128 checkerboards from one repeated PNG data URL and
readable Ukrainian text. The fixture selects the supplied `Go` font explicitly;
without that family, its first preview omitted text. No automatic font fallback
or all-CSS compatibility is claimed. Interlaced PNG, CSS background assets,
resource measurements, public-input isolation and physical image acceptance
remain open; no Pico or panel update was attempted.

## Decoder primary sources

- [Go image security considerations](https://pkg.go.dev/image#hdr-Security_Considerations):
  inspect dimensions with DecodeConfig before Decode; size rejection is the
  caller's responsibility.
- [PNG DecodeConfig](https://pkg.go.dev/image/png#DecodeConfig) and the installed
  Go 1.26.2 `src/image/png/reader.go`: direct PNG parsing and full decode.
- [Strict base64](https://pkg.go.dev/encoding/base64#Encoding.Strict): strict pad
  bits do not reject CR/LF, so this profile checks whitespace separately.
- [Group composition](https://www.w3.org/TR/compositing-1/#groupcompositing):
  preserve alpha until composition against the actual scene.
