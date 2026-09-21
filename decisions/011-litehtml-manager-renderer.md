# ADR-011: Evaluate litehtml as the manager HTML/CSS renderer

## Status

Superseded by ADR-012. litehtml remains a measured fallback candidate if the
Blitz spike fails its correctness, security, resource, or energy gates.

## Date

2026-09-03

## Context

The authoring format must be useful beyond one dashboard. It must accept normal
HTML with a practical CSS subset including classes and selectors, block and
inline layout, `inline-block`, `position: absolute`, text alignment, box model,
viewport-relative sizing, backgrounds, and images. Markdown or a project-only
scene language is too restrictive. Chromium/WebKit is unnecessarily broad.

## Decision

Use litehtml v0.10 as the first manager-side engine candidate. Pin the exact
version and hide it behind a Go `HTMLRenderer` provider. litehtml runs only on
the manager; TinyGo firmware continues to receive bounded resolved display-list
or one-bit bitmap operations.

The spike must use a thin, coarse-grained native boundary. It must not expose
litehtml C++ types to application code. Network access, navigation, events,
forms, and active content are disabled. HTML, CSS, fonts, and image assets enter
through bounded manager-owned inputs.

No production dependency is accepted until the spike proves the required
properties, deterministic 800x480 output, Ukrainian text, bounded failure on
malformed input, acceptable manager CPU/RAM/energy, and a viable sandbox or
equivalent containment strategy for native parsing.

## Why litehtml

- It is an embeddable HTML5/CSS2/CSS3 layout engine rather than a browser.
- v0.10 directly represents block, inline, inline-block, table and flex display;
  static, relative, absolute and fixed positioning; text alignment; viewport
  units; backgrounds and images.
- It supports classes, IDs, attribute selectors, combinators, embedded styles,
  and media features.
- Painting is delegated to `document_container`, allowing an e-paper-specific
  backend instead of a browser window or JavaScript engine.
- It is active, BSD-3-Clause licensed, and uses Gumbo under Apache-2.0.

## Consequences

- The manager build gains a C++17/native component and either CGO or a contained
  local renderer boundary.
- Font metrics, text painting, image loading/cache, clipping, backgrounds, and
  borders remain our backend responsibility.
- Complex arbitrary websites are not promised; the public contract is the
  tested litehtml subset and explicit resource limits.
- Server-side pixel or object diff remains separate from HTML layout and Pico
  panel-refresh policy.

## Sources

- https://github.com/litehtml/litehtml
- https://github.com/litehtml/litehtml/releases/tag/v0.10
- https://github.com/litehtml/litehtml/wiki/How-to-use-litehtml
- https://github.com/litehtml/litehtml/wiki/document_container
- https://raw.githubusercontent.com/litehtml/litehtml/v0.10/include/litehtml/types.h
