# ADR-013: Own the bounded HTML engine in Go

## Status

Accepted direction, 2026-09-06. Implementation and acceptance are tracked in
`../tasks/pure-go-engine.md`. The user authorizes research, necessary design
corrections, migration and non-Go renderer removal. Manual panel checks are
deferred until the complete firmware is ready. This is not hardware acceptance.

## Context

Blitz gave us real text/image output, but added Rust/Stylo, a process protocol,
two DOM interpretations and known layout bugs. The user now selects a bounded
HTML profile with inline `style` only, a pure-Go manager, and TinyGo on Pico.
Checkpoint `37a1471` preserves the previous implementation and its evidence.

## Decision

The manager owns HTML parsing, typed style validation, layout, shaping, assets,
painting, monochrome conversion, canonical scenes and confirmed output. Pico
receives validated resolved pixels, not HTML, CSS, fonts or source images.

Use existing `golang.org/x/net/html` v0.58.0 for the HTML5 tree,
`github.com/tdewolff/parse/v2/css` v2.8.16 for inline declaration syntax, and
evaluate Canvas at `v0.0.0-20260901160717-dae8cd8e19a7` for CPU painting and
typography. Pin dependencies and verify `CGO_ENABLED=0` before acceptance.

Pure Go means the selected runtime/build-tag closure needs neither Cgo nor
external rendering executables/interpreters. Never enable Canvas's optional
`latex`, `harfbuzz` or `fribidi` build tags; no LaTeX API or command execution is
part of the adapter. Default Go shaping/rasterization is qualified separately
from optional upstream GUI, GPU and export integrations.

Canvas is MIT, but the selected `canvas/text` closure imports
`github.com/benoitkugler/textprocessing/fribidi@v0.0.6`, whose package LICENSE
is LGPL-2.1. This is not an MIT-only dependency closure. The dependency is
manager-side only, never compiled into Pico. Source distributions must retain
third-party notices/licenses; before publishing manager binaries, provide the
corresponding dependency source and a reproducible relinking/rebuilding route
and review applicable LGPL obligations. This is a release constraint, not a
legal assurance. Track the inventory in `docs/engine-dependencies.md`.
Canvas is not an HTML/CSS layout engine. We own the bounded layout semantics.

| Component | Owns | Does not own |
| --- | --- | --- |
| Display profile | Logical size, orientation, palette, limits, target identity | HTML or register commands |
| HTML adapter | One parsed tree, element/attribute allowlist, node/depth budgets | CSS recovery or external fetching |
| Inline style model | Typed values, inheritance, shorthand resets, diagnostics | Stylesheets, selectors, JavaScript |
| Layout | Block/inline flow, wrapping, sizing, containing blocks, overflow | Transport or HAT pins |
| Assets | Bounded decode, immutable identity, intrinsic size and alpha | Unrestricted URLs or Pico decoding |
| Painter | Fonts, clips, stacking contexts, group opacity, final RGBA | Scheduling or hardware refresh |
| Mono/diff | Fixed viewport, palette conversion, deterministic damage | CSS or physical partial guarantees |
| Manager | Canonical scene, optional batching, current/pending/confirmed target | Driver implementation or public device key |
| Screen protocol | Dimensions, epoch/intent, bounded chunks, integrity, results | HTML, CSS or panel registers |
| Device runtime | USB priority, provisioning, authenticated network owner | Layout, source assets or full-frame storage |
| Panel/HAT/MCU | Controller lifecycle / electrical signals / TinyGo bindings | API, authentication or rendering |

Build direction: profile + parsers → layout + assets + painting → mono target
→ manager → USB/encrypted link → bounded device receiver → panel adapter.
Reuse `display`, `renderdiag`, `rasterasset`, `renderdiff`, `renderbatch`,
`refreshstamp`, credential/security modules and verified `panel.Stream`.
Create no empty packages merely to mirror this table.

## Corrections to earlier requirements

- ADR-011/012's renderer choice and stylesheet support are superseded. Reject
  `<style>`, external CSS and active content; `class` is metadata, not a selector.
- A bitmap marker cannot make unsupported CSS work. Custom rendering enters as
  a bounded image asset; do not retain a hidden Blitz/browser fallback.
- Server composition resolves transparency/z-order. Pico applies final opaque
  pixels. Retire the older requirement for Pico DOM/layout/full framebuffer.
- Adaptivity means layout at the target viewport, not stretching an 800×480
  screenshot. A new panel still needs a validated panel driver and capability.
- Controller-RAM streaming requires two ordered complete logical passes on the
  verified panel. Pixel diff does not by itself permit random SPI windows or
  partial refresh. Negotiate only implemented, verified operations.
- Preserve full refresh as the safe hardware baseline; partial remains gated.
  Software can be ready for manual acceptance without claiming ghosting, WPA3
  interoperability or energy measurements it has not observed.
- Keep safety/quality limits. Do not inherit the Blitz test skips into our
  engine: port their expected behavior into active Go regression tests.
- Non-Go removal covers this product's Rust worker, IPC adapter and associated
  runtime/build integration. Keep documentation, HTML/image fixtures, shell
  tooling and independent C/reference experiments; TinyGo/SDK are toolchains,
  not evidence that application logic is written in another language.

## Rendering invariants

One immutable scene and asset snapshot produces each target. Compute CSS lengths
in logical pixels; convert explicitly to Canvas coordinates/resolution. Bound
dimensions before allocation. `%` and viewport units use their documented
reference; they are not synonyms. An absolute child's containing block is the
nearest positioned ancestor, not an arbitrary static wrapper. Paint nested
stacking contexts atomically, preserving equal-z tree order. Group opacity is
applied once after children, with source-over against the actual backdrop.

Clip after geometry resolution; image `cover`/`contain` and position share one
fit calculation. Convert to monochrome after composition, using display-anchored
dither. Every damage rectangle comes from the same final target, including
erased old pixels and alignment expansion. Overflow and missing glyphs produce
bounded source-free diagnostics, not silent illegible downscaling.

The protected bottom-right timestamp represents the start of a successfully
completed full refresh. Partial preserves its pixels. The manager owns the
configurable 600-second maintenance schedule; no periodic Pico wake/poll.
Unknown/rejected/failed completion cannot advance the confirmed baseline/time.

## Sources and evidence boundaries

- HTML5 parsing and implicit/reparented nodes:
  https://pkg.go.dev/golang.org/x/net/html@v0.58.0#Parse
- Inline parser (`isInline=true`), parse errors and token values:
  https://pkg.go.dev/github.com/tdewolff/parse/v2/css@v2.8.16#NewParser
- Canvas CPU raster dimensions and compositing:
  https://github.com/tdewolff/canvas/blob/dae8cd8e19a7/renderers/rasterizer/rasterizer.go
- CSS flow/containing blocks:
  https://www.w3.org/TR/CSS22/visuren.html
- Box dimensions: https://www.w3.org/TR/CSS22/visudet.html
- Paint order: https://www.w3.org/TR/CSS22/zindex.html#painting-order
- Group opacity: https://www.w3.org/TR/css-color-3/#transparency
- Image fitting: https://www.w3.org/TR/css-images-3/#the-object-fit
- Decoder resource boundary: https://pkg.go.dev/image#hdr-Security_Considerations

These define semantics, not a claim of browser conformance. Exact supported
values, deviations, resource measurements and executable tests belong in the
engine specification. Canvas availability is verified; its acceptance gate is
still pending. No performance/energy improvement is asserted without measurement.
