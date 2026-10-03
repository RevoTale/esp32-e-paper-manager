# Lightweight manager renderer research

Status: architecture input. Candidate performance and energy claims are not
accepted until measured with this project's fixtures.

2026-09-06 follow-up: [pure-Go renderer and protocol research](pure-go-renderer-and-protocol-research.md)
compares six source-inspected candidates, supporting libraries and existing
formats. It records concrete CSS/security/resource gaps; it does not replace
the selected Blitz spike, firmware, or the accepted authoring contract.

## Decision

The authoring contract is normal HTML with a useful, tested CSS subset. A custom
viewport language and Markdown were rejected because they are less convenient
for general users. Chromium/WebKit remains too broad. Blitz `0.3.0-beta.1` is
selected for an isolated implementation spike under ADR-012; production
adoption still requires measurement and containment review.

The renderer should emit resolved display-list objects directly. Raster-only
content should be painted by rows or bounded tiles into one-bit output rather
than through a persistent full-page RGBA surface. At 800 x 480, an RGBA surface
is 1,536,000 bytes while a one-bit frame is 48,000 bytes, before allocator and
object overhead.

This is an architectural expectation, not proof of lower whole-system energy.
Cold start, CPU time, allocations, peak RSS, output bytes, and energy must be
measured on the actual manager target before accepting a heavier dependency.

## Candidates

### Existing project renderer — compatibility baseline

- Pure Go, bounded input profile, no browser process or JavaScript.
- Can produce stable display-list objects and one-bit bitmap tiles directly.
- Keeps unsupported behavior explicit and testable.
- Does not provide the general HTML/CSS authoring experience now required.

### litehtml v0.10 — fallback candidate

litehtml is a C++17 embeddable HTML/CSS layout engine. v0.10 directly includes
`block`, `inline`, `inline-block`, table and flex display modes; relative,
absolute and fixed positioning; text alignment; viewport units; background
images, sizing and positioning; CSS classes and broad selector support. It
deliberately delegates painting, font metrics, and image handling to a
`document_container`. That is a good architectural match for an e-paper backend,
but it introduces native code and requires our own bounded painter/cache.

The official project warns that it is not fully browser compatible. A reported
2026 issue measures very slow layout for a complex live GitHub page, so the
product must promise a tested profile rather than arbitrary websites. Our small
static dashboard fixtures require their own measurements.

Source: https://github.com/litehtml/litehtml
Source: https://github.com/litehtml/litehtml/releases/tag/v0.10
Source: https://raw.githubusercontent.com/litehtml/litehtml/v0.10/include/litehtml/types.h
Known complex-page risk: https://github.com/litehtml/litehtml/issues/451

### Dioxus Blitz 0.3.0-beta.1 — selected spike candidate

Blitz targets modern HTML/CSS with Stylo, Taffy, Parley, html5ever, and modular
painting. Its stated layout goals include flexbox, grid, tables, block, inline,
and absolute/fixed positioning. The user selected this broader modern layout
surface over litehtml.

The engine is explicitly beta, currently has no non-Rust bindings, and brings a
substantially larger Rust dependency graph. Integrate it as a pinned isolated
renderer process behind the Go provider. Use the narrow modular crates with
default features disabled; do not include the optional network, shell,
accessibility, tracing, or SVG paths unless a later measured requirement accepts
them.

Source: https://github.com/DioxusLabs/blitz
Source: https://github.com/DioxusLabs/blitz/blob/main/Cargo.toml
Source: https://github.com/DioxusLabs/blitz/blob/main/packages/blitz/Cargo.toml
Source: https://github.com/DioxusLabs/blitz/blob/main/packages/blitz-paint/Cargo.toml

### NetSurf LibCSS — parser/selector reference only

LibCSS is a portable C CSS parser and selector engine with low-memory goals. It
does not provide the complete HTML layout and paint pipeline, so adopting it
would still require the remaining NetSurf-style C stack or our own layout and
painting layers. It is useful as a standards and test reference, not the manager
foundation.

Source: https://www.netsurf-browser.org/projects/libcss/

### go-webengine/engine — prototype candidate only

This project is pure Go with `CGO_ENABLED=0` and claims HTML/CSS layout, RGBA
painting, images, SVG, and an option to disable JavaScript. Its own published
benchmarks show highly workload-dependent results and incomplete rendering
fidelity. The repository currently exposes no stable release, so its API,
security, correctness, and resource behavior require an independent audit and
benchmark before any code is reused.

Source: https://github.com/go-webengine/engine

### tdewolff/canvas — paint reference only

Canvas is a capable Go vector/raster library, not an HTML/CSS layout engine. It
may inform font and shape work, but importing it is justified only if measured
against the existing one-bit painter for binary size, allocations, and CPU.

Source: https://github.com/tdewolff/canvas

## Required benchmark before changing the baseline

Use sanitized representative fixtures: block/inline/inline-block cards,
absolute overlays, text wrapping, images, and one marked custom subtree.
Measure:

- cold-start and steady-state wall/CPU time;
- allocations, peak RSS, binary or container size;
- bytes emitted for initial and incremental updates;
- exact one-bit golden output and supported-profile diagnostics;
- energy on the actual always-on manager host, when its measurement interface
  is available.

Compare pinned Blitz against litehtml v0.10, the existing compatibility
renderer, and a full-frame baseline. Production adoption requires correct
output plus acceptable measured manager resource and energy cost; the words
"modular" and "lightweight" are not evidence by themselves.
