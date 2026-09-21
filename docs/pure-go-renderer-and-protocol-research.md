# Pure-Go server rendering and screen protocol research

Date: 2026-09-06. Status: research and shortlist, **not an accepted renderer
migration, new wire format, performance result, or hardware acceptance**.

## Outcome and scope

The manager can remain Go and the device TinyGo without writing a complete HTML
engine ourselves. There are real Go HTML-to-image implementations, not only
GUI toolkits. None of the reviewed solutions is a verified drop-in replacement
for our complete HTML/CSS, streaming, security and e-paper lifecycle contract.

**Recommendation:** test Ogre for the deliberately limited/template-oriented
profile, go-webengine for broader HTML behavior, and webrender when an
output-independent paint interface matters more than minimum integration work.
Keep gogpu/ui as the explicit HTML-to-widgets alternative, not an HTML engine.
Do not pick a winner from screenshots, stars, advertised coverage or language.

This research changes no firmware, dependency, container, wiring, refresh limit,
accepted ADR or test skip. Preserve the successful S6 bitmap replay and existing
recovery artifacts. Render-engine selection does not diagnose the earlier
live-manager delivery discrepancy.

The current reference remains [ADR-012](../decisions/012-blitz-manager-renderer.md).
[ADR-010](../decisions/010-viewport-first-markup.md) remains historical/rejected:
this report does not silently replace normal HTML with a proprietary layout DSL.

## Evidence and meaning of pure Go

- **Fact / source:** read upstream documentation, selected implementation paths,
  go.mod files, GitHub commit and repository metadata. Six core candidates were
  inspected at the commits below; ancillary candidates were screened at the
  documentation level.
- **Inference:** integration fit, maintenance risk and expected optimization
  opportunities. These are not measured improvements.
- **Unknown:** candidate build closure on our target, CSS acceptance, allocations,
  latency, peak RSS, TinyGo codec size and energy. No candidate was installed,
  built, benchmarked or sent to the panel during this research.
- Pure Go means Go implementation of application layout/painting/codec, without
  a hidden C/Rust renderer executed through CGo, dynamic FFI, a subprocess or
  embedded WASM. This does not claim the Go/TinyGo compiler, OS, MCU ROM or radio
  vendor firmware is itself written in Go.
- `CGO_ENABLED=0` is a useful gate, not proof of the previous condition.
  [resvg-go](https://github.com/kanrichan/resvg-go) explicitly uses WASM/resvg.
  [wgpu](https://github.com/gogpu/wgpu) has distinct Go, Rust-FFI and browser
  backends. Inspect the selected package/build-tag dependency closure.
- A module's go.mod includes optional/example/platform dependencies. Their
  presence alone does not prove they are linked into the headless executable.
  Conversely, a Go API alone does not prove its implementation is Go.
- Go-server compatibility does not imply TinyGo compatibility. The renderer and
  fonts stay on the manager; only bounded protocol/display packages need both.

### Reproducible source snapshots

Commit dates below are not release dates; repository age is not a correctness
score. Repository metadata showed none of these six archived on the check date.

| Candidate | Inspected commit | Commit date UTC | Root license / declared Go |
| --- | --- | --- | --- |
| Ogre | `32f3fad427d11973234c49f76613102216b1b225` | 2026-09-05 | MIT; Go 1.25.0 |
| go-webengine | `4fe1e1f6b871ee296271d6b3354969310e3e55f6` | 2026-09-06 | BSD-3-Clause; Go 1.26.4 |
| webrender | `5022e53cd46092a2054e80c2f5f990f9bad545d7` | 2026-03-04 | BSD-3-Clause; Go 1.23.0, toolchain 1.24.1 |
| gogpu/ui | `4d78f0d03a87c5c48bb66d3dc34f625de43067d0` | 2026-08-24 | MIT; Go 1.25.0 |
| Cogent Core | `fba74dd02245aa93ace50872d96da5e6b024fb33` | 2026-09-03 | BSD-3-Clause; Go 1.25.6 |
| gowkhtmltopdf | `0e72cdc20c93fd17cdd859ff339639fff28aedde` | 2026-08-30 | MIT; Go 1.26, toolchain 1.26.4 |

Root license labels are not a transitive license audit. Fonts, examples and
optional shaping/native backends require separate attribution checks.

## Candidate comparison

| Candidate | Provides | Fit and remaining cost |
| --- | --- | --- |
| [Ogre][ogre] | HTML + inline CSS / Tailwind-like classes → SVG, PNG, JPEG | Closest narrow ready-made template renderer; concrete display/opacity gaps below prevent unconditional adoption. |
| [go-webengine][engine] | HTML/CSS → layout → RGBA; optional JS/browser behavior | Broadest ready-image candidate examined; disable active behavior, cap canvas before allocation, audit young codebase. |
| [webrender][webrender] | Static HTML/CSS/SVG layout + abstract paint backend | Closest to server-resolved paint API; requires a real raster/backend adapter, not just a wrapper call. |
| [gogpu/ui][gogpu] | Typed widgets/layout + offscreen RGBA | Good explicit custom-profile route; HTML/CSS semantics and mapping still belong to us. |
| [Cogent Core htmlcore][htmlcore] | Existing HTML/Markdown → widget tree, offscreen platform | Actual ready HTML adapter; larger GUI lifecycle and styling semantics need containment and acceptance. |
| [gowkhtmltopdf][gowk] | Go HTML-template engine, PDF and PNG/JPEG | Real implementation, not wkhtmltopdf CLI wrapper; print-oriented and documents limited stacking/flex/grid semantics. |
| [tdewolff/canvas][canvas] | Paths, text layout/shaping, images, SVG/PDF/raster targets | Reusable painter/backend, not an HTML box-layout engine; select Go paths and audit backend licenses. |
| [gogpu/gg](https://github.com/gogpu/gg), [fogleman/gg][gg] | 2D drawing APIs | Useful for charts/custom bitmap producers; no automatic CSS cascade or HTML layout. |
| [go-text/typesetting][typesetting] | Go font parsing/shaping/bidi infrastructure | Reuse typography; do not write Unicode shaping or move it to Pico unnecessarily. |
| [kjk/flex][flex] / [nilslice/flex](https://github.com/nilslice/flex) | Go ports of Yoga's flex algorithm | Layout only. Historical Yoga snapshots, not proof of current CSS behavior; still need inline text flow, paint, selectors and assets. |
| [oksvg + rasterx][oksvg] | Partial SVG rasterization in Go | Useful server asset path; explicitly not complete SVG or HTML support. |
| [Mycel/opossum][mycel] | Experimental Go browser | README admits float/flex stubs; Plan9/duit assumptions make it poor reuse for this manager. |
| [Gio headless][gio] | Operation-list → offscreen image | Useful architectural reference; inspected package uses GPU backend paths, not a proven dependency-free CPU HTML renderer. |
| [templ][templ] / [quicktemplate][quicktemplate] | Generate HTML from Go | Good authoring-side tools; they do not turn HTML into pixels. |
| [Satori][satori] | Restricted HTML-like/React input → SVG | Strong profile/design reference, but JavaScript/Yoga rather than the requested all-Go implementation. |
| [resvg-go][resvg] | SVG rasterization through a Go API | No CGo, but a WASM resvg engine; fails the strict implementation-language condition. |

Adjacent PDF-only/document packages surfaced in discovery, including
[papyrus](https://github.com/grahms/papyrus),
[puregopdf](https://github.com/puregopdf/puregopdf),
[htmlpdf](https://github.com/zlfzx/htmlpdf) and
[dmundt/layout](https://pkg.go.dev/github.com/dmundt/layout).
They were not deep-audited or shortlisted: a PDF/document output path is not
evidence of a bounded headless bitmap API. Likewise, an “HTML canvas backend”
may generate JavaScript for a browser rather than rasterize in Go.
Do not count these discovery hits as verified replacements.

### Ogre: strongest narrow-profile lead, with blockers

**Fact:** inspected [go.mod][ogre-mod] directly depends on x/image, x/net,
x/text and go-text/typesetting, with no listed native renderer/WASM runtime.
[render/png.go][ogre-png] actually draws using Go image/draw/vector and encodes
PNG. The initial concern that this might merely wrap resvg was not supported
by the inspected code.

The [CSS reference][ogre-css] covers viewport units, width/height, margins,
padding, alignment, relative/absolute positioning, background images,
object-fit/object-position and opacity. Current code also has grid, although
the inspected site's display table lags that addition.

**Source-level gaps, not completed runtime bug reproductions:**

1. [ParseDisplay][ogre-properties] supports flex/none/block/contents/grid; unknown
   values become block. `inline-block` is absent. A normal mixed inline paragraph
   is not something to assume from “supports HTML.”
2. No z-index field/parser was found in the inspected [ComputedStyle][ogre-style]
   and [style resolution][ogre-resolve] path. The PNG child traversal does not
   establish browser stacking-context semantics.
3. The resolver assigns parsed opacity directly, but PNG `renderNode` changes
   opacity zero to one. This is a concrete contradictory code path to reproduce
   with `opacity:0`; do not advertise this value as accepted.
4. `Renderer.Render` [automatically fetches missing Google fonts][ogre-api].
   [Image fetching][ogre-image] constructs an HTTP client, follows its normal
   redirects, uses `io.ReadAll` and a package-level cache in the inspected path.
   Do not expose it to untrusted HTML without an asset/network policy and limits.
5. Public rendering returns encoded bytes; PNG allocates a full RGBA canvas.
   A zero-copy image/display-list hook and incremental rendering are not proven.

Repository created 2026-04-09. A useful real candidate, not an excuse to remove
our required z-order, inline layout or adversarial rendering tests.

### go-webengine: broader HTML, larger surface

**Fact:** [engine.go][engine-code] exposes RenderHTML, image output, injected
HTTP client and `DisableJS`. JS is enabled by default. Root [go.mod][engine-mod]
includes goja, esbuild, browser HTTP and image/font packages; setting DisableJS
does not by itself remove those imports from the build.

Its README describes block/inline/flex/grid/table/position behavior and raster
assets. It fits familiar authored HTML more closely than a widget mapping, but
those advertised capabilities still need our exact fixtures.

**Important source finding:** `newCanvas` grows height to laid-out content
height, beyond the requested viewport. Cropping the returned image does not
prevent the preceding large allocation. A fixed 800×480 admission/budget strategy
must exist before this can be the public renderer.

Repository created 2026-08-04; current source changes rapidly. Broad features
and upstream fidelity/coverage reports are not a production maturity guarantee.
No conclusion that it is faster or more energy-efficient than Blitz was measured.

### webrender: closest abstract drawing backend

**Fact:** a Go port of WeasyPrint's static rendering approach, not Python at
runtime. [README][webrender] explicitly requires a higher-level output backend.
The inspected module includes flex/grid/absolute/inline layout and stacking
implementation/test files; these are evidence of implementation, not CSS parity.

[Canvas interface][webrender-canvas] includes groups, DrawWithOpacity, paths,
fonts/text, raster images and gradients. This is closer to the desired resolved
painting model than translating HTML to interactive widgets.

**Cost:** implement or reuse a conforming Go raster backend, correct text
placement, clipping and group composition. Mapping each callback directly onto
Pico SPI would be incorrect: groups and backdrop-dependent alpha require server
composition. Pagination also needs a deliberate fixed-viewport policy.
README warns about production use and breaking changes. Repository dates to
2021, but age alone does not certify it.

### gogpu/ui and Cogent Core: two different widget routes

**gogpu/ui fact:** [offscreen/renderer.go][gogpu-offscreen] creates a Go drawing
context, lays out a widget and draws its tree into an RGBA result without an
application window. It does not parse HTML. The inspected offscreen Render
creates a new context; desktop dirty-region claims are not proof this path
incrementally reuses rendered pixels. [Module dependencies][gogpu-mod] include
GPU/FFI packages; check the actual selected closure, not the whole module label.

**Cogent fact:** [htmlcore.ReadHTML][htmlcore] already builds widgets from HTML.
Its [Context][htmlcore-context] accepts custom element handlers and resource
fetch callbacks; default fetching is http.Get. An [offscreen build tag][core-offscreen]
selects a [test/capture application][core-offscreen-app].
This is not the same as a stateless image function: global app state, event
loops and temporary app directories appear in the path. Full CGO-free closure
and exact CSS mapping remain unverified.

**Inference:** Cogent reduces adapter writing, but may bring more GUI behavior
than the task needs. gogpu/ui permits a more deliberate profile, at the cost of
owning that adapter. Neither automatically gives browser inline/stacking rules.

### gowkhtmltopdf: do not confuse engine and wrapper

**Fact:** the inspected project has its own HTML-template pipeline and
[ImageDocument API][gowk-api] with canvas dimensions and PNG/JPEG output.
It is not sebastiaanklippert/go-wkhtmltopdf, which wraps an external executable.

Its [compatibility matrix][gowk-css] explicitly limits flex/grid and calls
z-index ordering “lite,” not a complete CSS stacking-context tree. It documents
inline-block support as limited. Stronger for reports/tables than arbitrary
dashboard CSS. Source created 2026-08-03; README reports v0.2.5, while this audit
pins the commit rather than assuming every feature is in that release.

## Existing standards: use the correct layer

No reviewed specification supplies the whole HTML → Go compositor → TinyGo
receiver → authenticated atomic e-paper refresh system. “Standard” can mean
an authoring language, pixel file, serialization, transport or remote-display
protocol; these are not interchangeable.

| Standard / documented format | Useful role | Why it does not replace the entire design |
| --- | --- | --- |
| [HTML/CSS painting model][css-paint] | Familiar authoring, explicit supported profile, independent correctness oracle | Need an implementation and capability diagnostics. Parser success does not prove paint correctness. |
| [SVG / SVG Tiny 1.2][svg-tiny] | Standard server-side scene/asset interchange | Geometry, paths, text and composition still need rendering. “Tiny” is not proof of MCU fit. Do not put XML/CSS/font engines on Pico. |
| [PBM P4][pbm] | Simple raw monochrome fixture/export format | 1=black, MSB-first, packed rows match logical pixels well; no delta, authentication, transaction or refresh semantics. Normalize padding and bound header/comments. |
| [RFB / VNC, RFC 6143][rfb] | Prior art for changed rectangles, raw/RLE-like encodings and CopyRect | Informational RFC, not Internet Standards Track. Pixel format uses 8/16/32 bits per pixel; no native 1bpp raw pixel format. Some tile encodings pack palettes, so this is not a claim all RFB traffic is 8× larger. No e-paper lifecycle contract. |
| [CBOR, RFC 8949][cbor] + [CDDL, RFC 8610][cddl] | Compact typed metadata and an external schema | Encoding/schema only; must define framing, limits, ownership, commit and auth. Deterministic profile is explicit, not automatic. |
| [Protocol Buffers][protobuf] | Numbered fields, compact integers and bytes, generated contracts | Still requires our screen semantics; TinyGo runtime/codegen/size must be tested. No assumption protobuf requires gRPC. |
| [FlatBuffers][flatbuffers] | Direct access to serialized structures | Buffer offsets/random access are a poor default fit for bounded sequential streaming; “zero-copy” is not “no receive buffer.” |
| [CoAP, RFC 7252][coap] + [OSCORE, RFC 8613][oscore] | Established constrained-network transport/security option | Does not specify painting or physical refresh. Introducing another network stack needs a separate target/resource review; not required to replace the working USB path. |
| [A2UI v0.9][a2ui] / [DivKit][divkit] | Documented server-driven component descriptions/updates | Client builds/renders components; not resolved opaque rectangles. Putting their layout runtime on Pico reverses the agreed responsibility boundary. |
| [QOI][qoi] | Small RGB/RGBA image codec design | Not a dedicated packed monochrome format. Simplicity alone does not prove fewer bytes than 1bpp raw/RLE. |

**CBOR implementation caveat:** [fxamacker/cbor][fxamacker] has an experimental
TinyGo branch, explicitly not fuzz-tested, requiring a newer TinyGo development
fix than v0.41.1. This is not permission to add its normal Go runtime to Pico
unmeasured. A constrained typed codec can share test vectors/schema while server
and TinyGo use different compatible implementations.

**Recommendation, not a wire-format change:** use existing screen framing as
the baseline. Compare a restricted CBOR/CDDL metadata profile against its current
fixed binary encoding. Keep pixel bytes binary and bounded; do not base64 them.
PBM is attractive for portable captures/debug fixtures, not a mandatory live
transport replacement. If compression is added, raw must remain a bounded
fallback when compression expands data.

### E-paper projects as prior art

[TRMNL firmware][trmnl] demonstrates a device fetching a server-provided image
and update configuration; its documented algorithm sleeps and polls.
That behavior conflicts with the user's explicit always-reachable,
wait-for-server decision. Study the image/device boundary, not its sleep policy,
provisioning or hardware implementation.

[OpenEPaperLink][oep] is useful for display capability and constrained tag-transfer
research, not a ready Go/TinyGo stack for our Pico/HAT. Neither project's presence
means a universally interoperable e-paper renderer/refresh protocol exists.
Do not copy device register/timing/security assumptions across controllers.

## Architecture preserved by a server renderer replacement

```text
HTML + supported CSS / optional predefined classes
  → Go admission and bounded asset/font resolver
  → replaceable Go layout/paint provider
  → final composition (clip, stacking contexts, group alpha)
  → deterministic opaque monochrome target
  → existing damage/batching/transaction layer
  → USB first / authenticated network transport
  → TinyGo bounded receiver → panel adapter → controller RAM → refresh
```

A renderer's Go tree is not a wire format. Do not serialize Go pointers, widget
internals, generic gob values or renderer-specific structs into firmware.

Server-side full rasterization and full-frame transmission are different costs.
It is valid to render a coherent server target, compare packed pixels and send
only supported changed regions. Conversely, rectangle transport does not prove
the exact panel can safely perform a physical partial refresh.

At 800×480, arithmetic sizes are 1,536,000 bytes for one RGBA surface and
48,000 bytes for one packed 1bpp plane; 160×48 opaque pixels require 960 bytes
before headers. These are buffer/payload sizes, not process RSS or measurements.
The current two-pass streaming experiment can transmit 96,000 logical pixel
bytes for a full target. A renderer change does not remove this controller-pass
cost or prove the need for only one plane.

Follow [controller RAM streaming](controller-ram-streaming.md):
no full Pico framebuffer is reintroduced; do not assume controller readback,
CopyRect, glyph cache, arbitrary plane switching or retained RAM after sleep.
Negotiate only operations the concrete backend can implement correctly.

### Conflicts that every candidate must handle

Keep [render conflict invariants](render-conflicts-and-module-boundaries.md):

- Freeze one immutable scene/assets/fonts target per transaction. Group scene
  edits before rendering, not independently rendered overlapping patches.
- [Stacking contexts][css-paint] are not a global numerical z-index sort.
- [Group opacity][css-opacity] is applied to the composed group; overlapping
  children at 50% each are not equivalent to a 50% parent.
- Repaint exposed old bounds after move/delete/shrink. Changed backdrop can
  invalidate an unchanged transparent foreground.
- Resolve alpha against the actual background on the server. Do not flatten
  every subtree to white or send unresolved transparency to a write-only sink.
- Anchor dithering to absolute display coordinates; compare full and tiled
  results. Independent error-diffusion tiles can create seams and extra damage.
- Expanded byte-aligned rectangles must contain pixels from the same final
  target, including unchanged neighboring text and the protected timestamp.
- A failed renderer must return an error/diagnostic, not silently “fix” missing
  CSS by rasterizing with the same incorrect engine.
- Preserve generation/epoch, single-writer USB priority, complete-before-refresh,
  bounded powered waits and lost-ACK reconciliation. Received/committed is not
  independent proof that visible pixels changed.

Keep the configured last-successful-full-refresh corner and server-managed
maintenance schedule. Optional debounce/max_wait remain scheduling policies,
not display driver timing or a new wake/poll loop. No radio, WPA3, IPv6, security
or public exposure capability is gained merely by selecting a pure-Go renderer.

## Verification experiment to run only after approval

Do not rewrite three renderers simultaneously. Use one isolated submodule or
existing provider test harness, pinned dependencies, within the running Dev
Container. No new Pico firmware should be needed to compare server bitmap output.

1. **Cheap elimination:** compile chosen package with CGO disabled; inspect
   target dependency closure, embedded binaries/WASM and optional FFI tags.
   Render offline with all network denied; verify no hidden font/resource fetch.
2. **Profile corpus:** same sanitized fixtures, fonts, exact viewport and assets
   across the current Blitz baseline and candidate. Assert expected geometry
   as well as images; do not use Blitz's known-bug output as the oracle.
3. **Blocking tests:** inline-block beside mixed text; absolute child under a
   static ancestor; percentage/vw/vh; nested/negative/equal z-index; opacity
   0/0.5/1 and overlapping group children; transparent PNG over changing
   background; image contain/cover/position/clipping; Ukrainian and mixed text.
4. **Resource/abuse tests:** large dimensions, huge content height, deeply nested
   HTML, huge fonts/SVG/images, malformed input, missing resources, URL redirects
   and private destinations, cancellation and repeated renders. All work bounded.
5. **Repeatability/replay:** byte-identical packed output for unchanged input;
   full versus reconstructed patch target; move/delete erasure; stale/out-of-order
   render results; cache invalidation; timestamp region preservation.
6. **Measurements:** cold/warm wall and CPU time, allocs/bytes, peak RSS, idle CPU,
   binary/dependency size, raw/compressed bytes, unchanged-update cost. Count
   transfer passes/retries and controller powered time; measure actual energy
   separately. Do not infer energy from language or render latency alone.
7. **Hardware only after software gates:** replay one immutable candidate bitmap
   using the already successful direct USB workflow; match artifact hash and
   visible content. Keep receiver/firmware/wiring unchanged to isolate renderer.

For Ogre specifically, test the source-level inline/z-index/opacity findings
first. If the required profile fails, record the result and upstream issue/test
without disabling image support or expanding our own engine by stealth.
User-approved engine-bug skips apply only to the explicitly documented existing
cases, not every new candidate failure.

**Decision gate:** accept only an explicitly documented feature profile with
correctness/resource evidence. If narrowing that profile is necessary, ask the
user; do not silently drop existing requirements. Keep the current renderer and
protocol operational until a replacement passes. No requirement to copy a whole
library, introduce a GUI event loop, or rewrite TinyGo to change manager paint.

## Sources

Primary source files were read at the snapshots above. Moving docs below were
checked on the research date and must be rechecked before dependency adoption.

[ogre]: https://github.com/macawls/ogre/blob/32f3fad427d11973234c49f76613102216b1b225/README.md
[ogre-mod]: https://github.com/macawls/ogre/blob/32f3fad427d11973234c49f76613102216b1b225/go.mod
[ogre-api]: https://github.com/macawls/ogre/blob/32f3fad427d11973234c49f76613102216b1b225/ogre.go
[ogre-css]: https://github.com/macawls/ogre/blob/32f3fad427d11973234c49f76613102216b1b225/site/src/content/docs/reference/css.md
[ogre-properties]: https://github.com/macawls/ogre/blob/32f3fad427d11973234c49f76613102216b1b225/style/properties.go
[ogre-style]: https://github.com/macawls/ogre/blob/32f3fad427d11973234c49f76613102216b1b225/style/computed.go
[ogre-resolve]: https://github.com/macawls/ogre/blob/32f3fad427d11973234c49f76613102216b1b225/style/resolve.go
[ogre-png]: https://github.com/macawls/ogre/blob/32f3fad427d11973234c49f76613102216b1b225/render/png.go
[ogre-image]: https://github.com/macawls/ogre/blob/32f3fad427d11973234c49f76613102216b1b225/render/image.go
[engine]: https://github.com/go-webengine/engine/blob/4fe1e1f6b871ee296271d6b3354969310e3e55f6/README.md
[engine-code]: https://github.com/go-webengine/engine/blob/4fe1e1f6b871ee296271d6b3354969310e3e55f6/engine.go
[engine-mod]: https://github.com/go-webengine/engine/blob/4fe1e1f6b871ee296271d6b3354969310e3e55f6/go.mod
[webrender]: https://github.com/benoitkugler/webrender/blob/5022e53cd46092a2054e80c2f5f990f9bad545d7/README.md
[webrender-canvas]: https://github.com/benoitkugler/webrender/blob/5022e53cd46092a2054e80c2f5f990f9bad545d7/backend/graphics.go
[gogpu]: https://github.com/gogpu/ui/blob/4d78f0d03a87c5c48bb66d3dc34f625de43067d0/README.md
[gogpu-offscreen]: https://github.com/gogpu/ui/blob/4d78f0d03a87c5c48bb66d3dc34f625de43067d0/offscreen/renderer.go
[gogpu-mod]: https://github.com/gogpu/ui/blob/4d78f0d03a87c5c48bb66d3dc34f625de43067d0/go.mod
[htmlcore]: https://github.com/cogentcore/core/blob/fba74dd02245aa93ace50872d96da5e6b024fb33/htmlcore/html.go
[htmlcore-context]: https://github.com/cogentcore/core/blob/fba74dd02245aa93ace50872d96da5e6b024fb33/htmlcore/context.go
[core-offscreen]: https://github.com/cogentcore/core/blob/fba74dd02245aa93ace50872d96da5e6b024fb33/system/driver/driver_offscreen.go
[core-offscreen-app]: https://github.com/cogentcore/core/blob/fba74dd02245aa93ace50872d96da5e6b024fb33/system/driver/offscreen/app.go
[gowk]: https://github.com/chinmay-sawant/gowkhtmltopdf/blob/0e72cdc20c93fd17cdd859ff339639fff28aedde/README.md
[gowk-api]: https://github.com/chinmay-sawant/gowkhtmltopdf/blob/0e72cdc20c93fd17cdd859ff339639fff28aedde/documentation/library-api.md
[gowk-css]: https://github.com/chinmay-sawant/gowkhtmltopdf/blob/0e72cdc20c93fd17cdd859ff339639fff28aedde/documentation/compatibility-matrix.md
[canvas]: https://github.com/tdewolff/canvas
[gg]: https://github.com/fogleman/gg
[typesetting]: https://github.com/go-text/typesetting
[flex]: https://github.com/kjk/flex
[oksvg]: https://github.com/srwiley/oksvg
[mycel]: https://github.com/psilva261/mycel
[gio]: https://pkg.go.dev/gioui.org/gpu/headless
[templ]: https://github.com/a-h/templ
[quicktemplate]: https://github.com/valyala/quicktemplate
[satori]: https://github.com/vercel/satori
[resvg]: https://github.com/kanrichan/resvg-go
[css-paint]: https://www.w3.org/TR/CSS22/zindex.html#painting-order
[css-opacity]: https://www.w3.org/TR/css-color-3/#transparency
[svg-tiny]: https://www.w3.org/TR/SVGTiny12/
[pbm]: https://netpbm.sourceforge.net/doc/pbm.html
[rfb]: https://www.rfc-editor.org/rfc/rfc6143.html
[cbor]: https://www.rfc-editor.org/rfc/rfc8949.html
[cddl]: https://www.rfc-editor.org/rfc/rfc8610.html
[protobuf]: https://protobuf.dev/programming-guides/encoding/
[flatbuffers]: https://flatbuffers.dev/
[coap]: https://www.rfc-editor.org/rfc/rfc7252.html
[oscore]: https://www.rfc-editor.org/rfc/rfc8613.html
[a2ui]: https://a2ui.org/specification/v0.9-a2ui/
[divkit]: https://divkit.tech/docs/en/concepts/faq
[qoi]: https://qoiformat.org/qoi-specification.pdf
[fxamacker]: https://github.com/fxamacker/cbor/blob/master/README.md
[trmnl]: https://github.com/usetrmnl/trmnl-firmware
[oep]: https://github.com/OpenEPaperLink/OpenEPaperLink
