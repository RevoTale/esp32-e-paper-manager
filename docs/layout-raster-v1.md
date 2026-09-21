# Layout and monochrome raster profile v1

## Geometry

- Layout accepts displays from 128x64 through 4096x4096 pixels.
- The synthetic document root owns the whole logical surface.
- Block flow is vertical. `display:flex` supports bounded row or column flow;
  row gaps are removed before equal-width distribution.
- Font metrics and wrapping use the same deterministic glyph iterator as the
  rasterizer, so entities and collapsed HTML whitespace cannot diverge.
- Every node records `fit`, `wrapped`, `clipped`, `elided`, or `hidden`.
  Vertically overflowing `data-priority="low"` content is elided; other
  overflow is clipped. Unsupported priority values are rejected.

The bottom-right 88x16-pixel timestamp rectangle is firmware-owned. Layout
reports intersections as clipped and raster painting from HTML always excludes
the rectangle. Firmware clears it, adds a border, and writes an exact
`YYYY-MM-DD HH:MM` value after all document content.

## Raster contract

- Output is painted directly into the one caller-owned packed 1-bit frame.
- Element backgrounds, borders, normal/bold text, alignment, and ancestor
  overflow clips are supported. Hidden or elided ancestors suppress children.
- The pinned `golang.org/x/image/font/basicfont` X11-derived `Face7x13` mask is
  sampled directly without an intermediate image. Requested 8-32 pixel font
  heights are deterministically resampled.
- The initial font asset covers printable ASCII. Other UTF-8 code points render
  as the replacement glyph; this limitation is visible instead of silently
  dropping text.
- Drawing retains no input and makes no steady-state heap allocation.

## Evidence

The 128x64 golden frame SHA-256 is
`d4e9efc6300f46066a07df70f3dd671f99f1c734d6f34aa1e17ee7a065906402`.
Tests also run at 320x200, assert zero `Draw` allocations, and reject malformed
timestamp, storage, and layout contracts.

On TinyGo 0.41.1 for `pico2-w`, the full-device probe containing one 32 KiB
input, parser nodes, CSS rules/styles, layout boxes/outcomes, the existing
48,000-byte frame, and raster code uses 105,340 static RAM bytes and 49,028
flash bytes. This is a compile/resource result, not physical acceptance.

## References

- CSS Flexible Box Layout Module Level 1: https://www.w3.org/TR/css-flexbox-1/
- CSS Gap Decorations Module Level 1: https://www.w3.org/TR/css-gaps-1/
- CSS Overflow Module Level 3: https://www.w3.org/TR/css-overflow-3/
- Go `basicfont`: https://pkg.go.dev/golang.org/x/image/font/basicfont
