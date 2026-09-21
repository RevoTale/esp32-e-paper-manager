# Engine: bounded inline HTML rendering

2026-09-06. Implementation contract for ADR-013, not a browser-conformance claim.
Manager-only packages own parsing/layout/painting. The user approved autonomous
implementation and necessary corrections; manual panel acceptance comes last.

## Input and output

One UTF-8 HTML document/fragment and one trusted logical display profile produce
one independently owned RGBA target, diagnostics and canonical packed mono frame.
No host files, URL fetches, scripts, CSS stylesheets or ambient fonts are read
as a consequence of document input. An administrator selects embedded fonts.
Use one HTML5 parsed tree for validation and layout, never a second parser for
source ownership. Element ordinals include implied HTML/head/body nodes.

HTML tags: html, head, body, title, meta charset, div, section, article, main,
header, footer, aside, span, p, h1–h6, strong, b, em, i, small, br, hr, ul, ol,
li, figure, figcaption, img, pre, code. Global attributes: id, class (metadata),
style, lang, dir=ltr/rtl, title. Images also accept src/alt/width/height.
Unknown tags/attributes and duplicates reject; comments and doctype have no
paint effect. Reject scripts/forms/events/navigation, namespaces, SVG,
stylesheets and `<style>` even inside bitmap markers. A legacy bitmap marker
is admitted only as metadata; it grants no additional CSS or resource rights.

Image input initially supports bounded PNG and JPEG data URLs. Asset APIs may
pass already-validated immutable raster assets; arbitrary network URLs remain
disabled. Decode header/dimensions before allocating full pixels; scene-wide
limits apply even to repeated references. Invalid images reject the target.
`alt` describes the source; it is not permission to silently hide corrupt input.

## Limits

Keep existing 32,768-byte HTML, 2,048-pixel dimension and 1,048,576-pixel viewport
limits. Further manager bounds: 1,024 nodes, depth 32, 32 attributes/element,
4,096 declarations/document, 256 tokens/declaration, 32 images, 1,048,576 decoded
asset pixels/scene and 4 MiB aggregate RGBA image storage. Encoded assets count
towards HTML size. CSS values are finite; length operands have magnitude at
most 8,192; resolved coordinates have magnitude at most 32,768 logical pixels.
Font size 8–128 logical pixels.
Reject incompatible min/max and numerical overflow; do not silently clamp.

Limit painted pixel visits and temporary group/clip surfaces explicitly before
allocation, with 64 Mi pixel visits and eight simultaneous opacity groups as
initial manager ceilings. Bound tile/paint operations to 65,536 per scene, checked
before iteration even when a tile is subpixel or transparent. Temporary paint
surfaces have an aggregate 32 MiB ceiling, excluding the 4 MiB output and 4 MiB
decoded asset budgets; check actual allocations before creating a surface.
Reject zero or
non-progressing repeat periods. Bound glyphs by input text and work counters.
Cancellation is checked between bounded operations; a timeout is not a license
to change output semantics. Existing 5s rendering goal is now measured on the
manager separately from transport/SPI/BUSY. Target Pico budgets concern decoded
chunks and network state, not document parsing. Do not raise these limits to
make fixtures pass.

## Property families

Unsupported values return a stable source-free diagnostic with element ordinal,
decoded inline-style byte range and code. Unknown property is distinct from
malformed grammar and unsupported value. Whole-scene rejection is atomic.
Declarations apply in source order; important beats normal even if earlier.
Expand shorthands into their complete longhand effect before priority resolution.
Inheritance is explicit, not a copied entire parent style.

| Family | Supported values and semantics |
| --- | --- |
| display | block, inline, inline-block, none; element defaults follow semantic tag |
| position | static, relative, absolute; insets auto or bounded lengths |
| dimensions | width/height auto or nonnegative lengths; min width/height nonnegative; max width/height none or nonnegative |
| units | px, %, em, rem, vw, vh; unitless zero; font-size %/em against inherited size, root font-size rem against initial 16px; other em/rem against computed local/root font |
| box-sizing | content-box default, border-box |
| margin/padding | 1–4 values and each side; padding nonnegative; margins signed/auto where CSS permits; percentage spacing against containing width |
| border | uniform solid or none, width/color and per-side width/color/style; unsupported line styles reject |
| border-radius | nonnegative uniform radius; percentages resolve against box; complex elliptical syntax rejects |
| colors | named CSS colors, #RGB/#RGBA/#RRGGBB/#RRGGBBAA, rgb()/rgba(), transparent, currentColor |
| z-index | auto default or integer -32768…32767; positioned stacking contexts, not global node sorting |
| opacity | number 0…1; values below 1 create an isolated stacking group, applied once after descendants; 1 does not create a group |
| overflow | visible default, hidden or clip; no interactive scroll container |
| font-family | Go/sans-serif default, monospace; bounded administrator-owned fonts, no remote/system lookup |
| font-size/weight/style | inherited size (16px root), normal/bold or 400/700, normal/italic |
| line-height | normal, positive number or length; inherited computed rules documented in tests |
| text-align | inherited left/right/center/justify; direction affects start/end aliases |
| white-space | inherited normal/nowrap/pre/pre-wrap/pre-line; preserve relevant newlines/spaces |
| overflow-wrap | normal or anywhere; grapheme-safe wrapping, not byte splitting |
| text-overflow | clip or ellipsis on a constrained non-wrapping line; other combinations diagnose |
| background | transparent default; color, one image, size auto/contain/cover/length pair, position keywords/length pair, repeat/no-repeat/repeat-x/repeat-y |
| image | intrinsic ratio, explicit aspect-ratio auto or positive ratio; object-fit fill/contain/cover/none/scale-down; object-position keyword/length pair |

No selectors, flex/grid, table layout, floats, calc(), variables, filters,
gradients, transforms, animations or browser fallback. This is a documented
subset, not silent replacement with a flex-only or absolute-only layout engine.

## Layout and composition

Logical viewport is fixed; content never grows the output bitmap. `%` refers
to the property's containing-block dimension, vw/vh to the viewport. Image and
background positioning percentages use remaining space after sizing: a 20px
image centered in 100px starts at 40px, not 50px. Height
percentages with indefinite containing height follow CSS auto behavior, not a
percentage of whichever height was computed later. Absolute positioning uses
the nearest positioned ancestor's padding box, otherwise the viewport.
Paired auto insets use a bounded static-position estimate (block-flow cursor
or current paragraph origin), not a second hypothetical text-cursor layout.
Absolute descendants skip intervening overflow clips until their containing
block; non-replaced inline elements do not establish overflow clips.
Relative offsets move paint, not following siblings' normal-flow positions.
Relative top/bottom percentages against an auto/content-dependent height act
as auto; a fixed or resolvable percentage height is a valid base. Horizontal
relative overconstraint uses the containing block's direction, not the child's
dir override. Regression expectations are cross-checked with the
[WPT height matrix](https://github.com/web-platform-tests/wpt/blob/dabe52e02fe620e75664f5904fb2b90fd95ac77f/css/css-position/position-relative-015.html)
and [CSS relative positioning](https://www.w3.org/TR/CSS22/visuren.html#relative-positioning).

Block and inline flows use final font metrics, image dimensions and available
width. Resolve min/max, auto margins, padding, border and box sizing before
wrapping. Explicit profile deviation: no browser vertical margin collapsing;
adjacent block margins add, documented and tested. Inline-blocks are atomic
line items. Their visible-overflow baseline comes from the last in-flow line;
otherwise the bottom margin edge is used. Text/image overlap is composited before quantization. Negative
z-index belongs to a stacking context; descendants never escape that context.

Overflow outcomes are wrapped, clipped, elided or rejected. Never shrink below
8px to squeeze a document onto screen. Return diagnostics for clipped content
and missing glyphs. Bottom-right reserved timestamp geometry is accounted for
in composition/diagnostics and protected from absolute elements as well as normal
flow. `Options.Reserved` keeps only that rectangle white for the later stamp;
nonwhite overlap emits `reserved-overlap` with element zero (whole composition).
It does not reduce the viewport or consume a full bottom strip.

### Explicit combination boundaries

Canvas exposes one line-break policy per rich-text block. Changing `white-space`
inside a non-replaced inline run therefore rejects with that declaration's
diagnostic; use a block or `display:inline-block` for a different policy. Do not
silently wrap `nowrap` spans. Each advertised whitespace mode remains available
on its own block formatting context. Source: pinned Canvas `RichText.ToText`,
`text.GlyphsToItems`; [CSS text scope](https://www.w3.org/TR/css-text-3/#white-space-property).
Preserved TAB characters in pre/pre-wrap currently reject: the renderer does
not implement line-relative CSS tab stops, and replacing each TAB with eight
spaces is not equivalent. Supply explicit spaces. Tabs in collapsing whitespace
modes remain ordinary collapsible whitespace. Newlines and repeated spaces in
pre/pre-wrap remain supported.

Negative horizontal margins on non-replaced inline boxes reject because the
Canvas object-advance interface cannot faithfully represent them. Signed block
and inline-block margins retain their documented geometry. A block descendant
inside an inline box also rejects; anonymous inline splitting is not implemented.
Positive inline padding/border/margins advance content; vertical decorations
do not enlarge line-height. Fragments merge once per owner per line, avoiding
repeated alpha composition. Wrapped decorations are sliced: side edges/radii
are not cloned at line breaks, and backgrounds continue across the stitched
fragments. Physical ends follow the parent's inline progression. The detailed
baseline/fragment contract and source references are in
[native inline layout](docs/engine-inline-layout.md).
Multiline positioned inline ancestors use the
enclosing fragment rectangle, an explicit deterministic choice for CSS2.2's
undefined multiline case. Ellipsis preserves original line metrics even when
the large-font run is hidden: [paint-only requirement](https://www.w3.org/TR/css-overflow-4/#ellipsis-interaction).
Ellipsis currently accepts text-only runs, without inline objects or nonzero
horizontal inline padding/border/margins. Such combinations reject explicitly;
put the decoration on the containing block instead. Prefix elision must never
silently remove inline-object advances or change the painted box geometry.

Background positioning uses the padding box and painting the border box. Inner
rounded clips subtract each border/padding edge independently; square and round
borders use a non-overlapping ring so transparent corners are painted once.
Image sampling is CPU approximate bilinear, then display-anchored 4×4 Bayer
quantization. This is not pixel equality with Blitz's sampling/quantization.

### Deterministic defaults

The display backdrop is opaque white. Root/body have zero margin and padding;
the body background paints the body box (no browser canvas-background propagation).
All containers are block except span/strong/b/em/i/small/code/br/img (inline).
Head/title/meta do not paint. Base text is Go Regular, 16px, normal line height,
black, left aligned, LTR. Strong/b and headings are bold; em/i are italic.
Headings h1–h6 use 32/24/20/18/16/16px. Small is 0.8em, pre/code monospace;
pre preserves whitespace. Default p/headings have no implicit margins.
Lists use a 24px left indent, li block flow, a bullet or 1-based decimal marker
in that indent; nested lists number independently. No CSS list-style support.
HR has a 1px solid currentColor top border. These are engine defaults, not a
claim to reproduce a browser's user-agent stylesheet.

## Tests, commands and completion

From the existing Dev Container, experiment root:

```sh
CGO_ENABLED=0 go test ./engine
go test -race ./engine
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ./engine
./scripts/quality.sh task
./scripts/quality.sh full
```

Use RED→GREEN tests per feature. Geometry tests use independent expected boxes;
RGBA tests cover alpha/paint order before monochrome goldens. Fix font and
dependency versions. Inspect representative generated PNGs at multiple sizes.
Fuzz invalid HTML/CSS/assets under work budgets; fuzz replay and transport
separately. Benchmark cold/warm renderer memory and latency; include failed
scenes, not only successful small text. No cross-renderer pixel equality claim
for typography unless the font/shaping/antialiasing contract actually matches.

Engine completion requires every property family above either implemented with
tests or explicitly returned to the user as an unresolved requirement. The
Canvas adapter is not the finished engine. Final product also requires the
unified firmware and manager tasks in `tasks/pure-go-engine.md`.
