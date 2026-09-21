# Viewport layout markup v1

Status: rejected design retained for history. Normal HTML plus the tested CSS
profile in `manager-html-css-profile-v2.md` is the authoring contract.

## Example

```html
<screen version="1" design-width="800" design-height="480"
        fit="contain" background="white" overflow="error">
  <stack id="content" direction="column" left="24" top="20"
         right="24" bottom="40" gap="12" padding="8">
    <text id="title" size="28" weight="bold" align="center">
      Dashboard
    </text>
    <image id="status" src="asset:status" width="100%" height="240"
           fit="contain" image-position="center"/>
  </stack>
  <text id="updated" position="absolute" right="12" bottom="8"
        size="12" align="right">Updated 15:04</text>
</screen>
```

## Viewport

`screen` is the single root. `design-width` and `design-height` define logical
coordinates. The manager maps them to the provisioned physical display using:

- `fit="contain"`: preserve aspect ratio and letterbox;
- `fit="cover"`: preserve aspect ratio and clip;
- `fit="stretch"`: scale axes independently;
- `fit="none"`: one logical pixel is one display pixel.

`viewport-align` accepts `top-left`, `top`, `top-right`, `left`, `center`,
`right`, `bottom-left`, `bottom`, or `bottom-right`. The default is `center`.
Layout uses logical coordinates and rounds once when physical display-list
coordinates are emitted.

## Elements

| Element | Purpose |
|---|---|
| `screen` | Root viewport and page background |
| `stack` | Deterministic `row` or `column` child layout |
| `box` | Background, border, padding, clipping, and grouping |
| `text` | Bounded text layout and painting |
| `image` | Manager-decoded image asset |
| `spacer` | Fixed or remaining space within a stack |

Every renderable element may have a stable `id`. Duplicate IDs are invalid.
IDs identify objects across revisions and must not encode transient array
positions.

## Geometry and flow

Lengths accept non-negative integer logical pixels or percentages. Signed
integers are accepted only for `x`, `y`, and edge offsets. V1 has no `calc()`,
`em`, `rem`, `vw`, or `vh` grammar.

Common geometry attributes:

```text
x y width height min-width min-height max-width max-height
left right top bottom anchor position padding margin clip
```

- `position="flow"` is the default inside a stack.
- `position="absolute"` removes the element from stack flow.
- Edge attributes constrain an absolute element against its nearest containing
  `screen` or `box`.
- `anchor` uses the same nine values as `viewport-align` and interprets explicit
  `x`/`y` coordinates relative to that point.
- Conflicting or under-constrained geometry returns a typed error. The engine
  does not guess browser-style conflict resolution.

`stack` attributes:

```text
direction="row|column" gap align="start|center|end|stretch"
justify="start|center|end|space-between" padding
```

Children are processed in document order. `spacer grow="1"` shares remaining
space with other growing spacers. This is deliberately smaller than Flexbox.

## Text

```text
size weight="normal|bold" color="black|white"
align="left|center|right" vertical-align="top|middle|bottom"
line-height wrap="word|character|none" max-lines overflow="clip|ellipsis|error"
```

Fonts are manager-provisioned resources. An unknown font or missing glyph
returns a typed diagnostic or uses an explicitly configured fallback; it never
changes silently between hosts.

## Backgrounds and images

`screen` and `box` support `background="black|white|asset:ID"`,
`background-fit`, and `background-position`. `image` supports `src`, `fit`,
`image-position`, and clipping. The manager decodes and resamples source images;
Pico only receives validated one-bit pixels or cached asset references.

## Overflow

Each container selects `overflow="clip|error"`. `error` returns the element ID,
computed bounds, and viewport bounds. `clip` clips deterministically and records
the condition in render diagnostics. Silent accidental overflow is forbidden.

## Boundary behavior

- Unknown elements, attributes, enum values, duplicate IDs, invalid lengths,
  and conflicting constraints return stable typed errors with source location.
- Limits cover document bytes, depth, elements, text bytes, decoded image
  pixels, and final display-list operations.
- Scripts, event handlers, active content, CSS, and network fetching from markup
  are forbidden.

## Acceptance

- Golden tests cover every viewport fit/alignment combination at multiple panel
  sizes and rotations.
- Layout tests cover nested row/column stacks, padding, gaps, percentages,
  spacers, anchors, edge constraints, and absolute overlays.
- Overflow tests prove both exact diagnostics and clipping.
- Stable-ID tests prove that changing one element emits only its old/new dirty
  bounds and required display-list patch.
