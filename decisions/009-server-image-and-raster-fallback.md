# ADR-009: Render images and custom HTML on the manager

## Status

Accepted direction. Exact source codecs, fetch policy, asset limits, and the
server rasterizer implementation remain contract and measurement work.

## Date

2026-09-02

## Context

ADR-008 moves document parsing and layout from Pico to the Go manager. ADR-011
selects normal HTML and a tested CSS subset; ADR-012 selects pinned Blitz for
the renderer spike. A useful dashboard still needs substantial styling, the
HTML `img` element, and an escape
hatch for custom static content. Requiring Pico to decode PNG, JPEG, SVG, fonts,
or general CSS would consume flash, RAM, CPU, radio time, and attack surface.
Sending every page as a complete bitmap would work but discard object-level
diffs and normally send more data.

## Decision

The manager-facing HTML adapter has two explicit rendering paths:

1. The native resolved-render path supports the bounded contract in
   `docs/manager-html-css-profile-v2.md`. It emits text, geometry, asset, and
   bounded bitmap display-list operations.
2. An explicit server-raster fallback renders a marked custom subtree into its
   final clipped one-bit pixels. Pico receives bounded `BLIT_1BPP_RAW`
   operations and never sees or interprets the original unsupported markup.

Both paths use the bounded manager renderer. A persistent headless Chromium,
WebKit, JavaScript VM, or general browser networking stack is not part of the
default architecture. A third-party renderer may be introduced only after the
project benchmark in `docs/lightweight-renderer-research.md` proves its resource
cost and output behavior on representative fixtures.

Support `<img>` in the manager profile. The manager validates and decodes the
source, applies intrinsic sizing, CSS sizing, `object-fit`, `object-position`,
clipping, alpha composition, and monochrome conversion, then emits a one-bit
raw bitmap or a negotiated cached asset. Pico does not decode the source image
format.

Raster fallback is explicit, initially through
`data-epaper-render="bitmap"`. Unsupported HTML or CSS outside a marked fallback
returns a typed diagnostic. It must not be silently approximated. JavaScript,
event handlers, navigation, forms, and active content remain forbidden in both
paths; bitmap fallback is not a browser-execution escape hatch.

Image acquisition is separate from layout semantics. Uploaded/content-addressed
assets and bounded data images may be supported. Remote URL fetching is disabled
until an administrator enables a policy with scheme/host allowlists, DNS and IP
validation, redirect limits, byte/pixel/time limits, MIME verification, and
protection against private-network requests.

The direct USB compatibility tool may accept the same HTML only by running the
manager renderer on the host and sending its display list. The older on-device
HTML profile remains image-free and does not gain image decoders.

## Consequences

- Common HTML remains object-diffable and bandwidth-efficient.
- Custom static output remains possible, with bandwidth proportional to the
  rasterized dirty area rather than HTML complexity.
- Exact source codecs and CSS coverage are manager capabilities negotiated and
  tested independently from the Pico display-list version.
- A changed image or raster subtree can fall back to raw bytes; repeated images
  should use content-addressed assets when that is smaller and cache-safe.
- Server raster output must be deterministic for a display profile, renderer
  version, fonts, image decoder versions, dithering mode, and input assets.
- Rasterization should paint directly to one-bit rows or bounded tiles when
  practical; a persistent full-page RGBA canvas is not the default.
- Tests must cover corrupt and oversized images, decompression bombs, alpha,
  aspect ratio, clipping, monochrome conversion, fallback boundaries, SSRF
  policy, cache invalidation, object diffs, and exact Pico bitmap output.

## Sources

- HTML `img` element: https://html.spec.whatwg.org/multipage/embedded-content.html#the-img-element
- CSS Images, including `object-fit` and `object-position`: https://www.w3.org/TR/css-images-3/
- Lightweight renderer research: ../docs/lightweight-renderer-research.md
