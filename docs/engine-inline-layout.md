# Native inline layout

This documents the pure-Go manager engine, not Blitz or Pico rendering. The
existing accepted property set and resource limits remain unchanged.

## Inline-block baseline

A visible-overflow inline-block aligns its last in-flow line baseline with the
surrounding line. In-flow block descendants contribute their last line; a later
empty block does not erase it. Outside list markers and absolute descendants do
not replace it. An inline-block without a line, a replaced image, or an
inline-block with non-visible overflow uses its bottom margin edge instead.
The visible-overflow baseline may lie beyond an explicitly smaller height.
Source: [CSS 2.2 line height and vertical alignment](https://www.w3.org/TR/CSS22/visudet.html#leading).

Canvas still supplies shaped glyphs, atomic object widths and line breaks. The
engine records a border-relative baseline and uses it in the existing final
line-metric and object-placement pass. Relative movement remains a later paint
offset and does not alter normal-flow baseline selection. No dependency patch
or second text shaper is involved.

## Sliced decoration

Wrapped non-replaced inline decoration uses the default sliced behavior. Side
borders, padding and margins occur at the unbroken ends, not at every line
break. The parent's inline progression selects those physical ends, including
asymmetric RTL edges. Top and bottom decoration continues along each fragment;
broken corners are not rounded. Source: [CSS 2.2 inline formatting](https://www.w3.org/TR/CSS22/visuren.html#inline-formatting).

The painter joins fragment widths into virtual horizontal geometry and clips
each draw back to its actual fragment. Background positioning therefore
continues across lines instead of restarting on each line. Each fragment keeps
its own height; an aspect-derived image width uses the tallest fragment's height
and retains that size while being positioned in each fragment. Sources:
[fragmented decoration](https://www.w3.org/TR/css-break-3/#break-decoration) and
[joining sliced boxes](https://www.w3.org/TR/css-break-3/#joining-boxes).

No strip image or full intermediate canvas is allocated. Existing viewport,
ancestor clips, temporary-memory and raster-operation budgets still bound the
work; background tile enumeration also intersects the actual fragment clip.
Zero-sized no-repeat background images remain unpainted.

Atomic inline objects contribute width to their surrounding inline owner's
fragment; the owner retains its font-metric decoration height. `overflow`
does not create a clipping box on non-replaced inline elements. Absolute
subtrees skip overflow clips between themselves and their containing block,
but still respect that containing block's clip and the fixed viewport.
Source: [CSS overflow applicability](https://www.w3.org/TR/CSS22/visufx.html#overflow).

## Pinned Canvas object-span boundary

Canvas `dae8cd8e19a7`, `RichText.ToText`, can merge post-newline text into a
same-face object span. `TextSpan.IsText` then returns false and `RenderTextTo`
does not paint those glyphs. A direct dependency reproduction remains active
beside a corrected-control test. Source: [pinned text implementation](https://github.com/tdewolff/canvas/blob/dae8cd8e19a7/text.go).

The adapter assigns each object run a copied `FontFace` descriptor. Text and
object runs retain distinct identities, unchanged font metrics and the same
font/shaper references. Each descriptor maps back to its original element owner;
the renderer's existing gate still serializes the mutable font caches. This
adds one descriptor per object, not a font copy or bitmap. At most two synthetic
decoration boundaries per inline element, or one atomic object, are created;
the scene's 1,024-node bound limits their number. Re-evaluate the reproduction
and adapter when changing the Canvas pin. No upstream code is patched.

For absolute descendants whose paired insets are auto, the engine uses a
bounded static-position estimate: the current block-flow cursor or, inside an
inline run, the current paragraph origin. It does not construct a second
hypothetical layout to locate the exact text cursor. CSS 2.2 permits such an
estimate; authors needing exact placement should specify an inset on each axis.
Source: [absolute width/static position](https://www.w3.org/TR/CSS22/visudet.html#abs-non-replaced-width).

This does not add `box-decoration-break`, arbitrary writing modes, anonymous
block-in-inline splitting, or a new bidi-fragment model. The documented
one-fragment-per-owner-per-line model remains; this is not full browser CSS
conformance. Host geometry and exact RGBA tests are software evidence, not
physical display acceptance.

## Checks

Run inside the existing project Dev Container:

```sh
go test -race ./engine/...
tools/bin/golangci-lint run ./engine/...
```

The regression fixtures cover last/empty/nested/list/overflow baselines, sibling
pixels, LTR/RTL side edges, broken radii, continuous backgrounds and unequal
fragment heights, object-owned fragments and exact post-break glyph pixels.
The compact `engine/testdata/viewport-review.html` fixture is exercised at
250×122, 400×300, 800×480 and 480×800 with stable, independently owned output.
Existing whole-engine and resource-budget tests remain active.
