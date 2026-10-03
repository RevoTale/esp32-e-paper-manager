# Bounded dashboard CSS profile v1

> This is the older direct on-device/USB compatibility profile. The broader
> manager-owned profile is specified in `manager-html-css-profile-v2.md`.

CSS is optional and deterministic. The device accepts up to 64 simple rules
and 512 declarations in total, including inline `style` attributes. It does not
implement browser stylesheets, layout quirks, media queries, or dynamic state.

## Selectors

One simple selector per rule is supported:

- element name, for example `section`;
- one class, for example `.card`;
- one ID, for example `#status`.

Combinators, selector lists, compounds, pseudo-classes, pseudo-elements, and
attribute selectors are rejected. Specificity is `ID > class > element`; later
rules win ties. Inline style has highest specificity. Class and ID values are
case-sensitive; HTML tag matching is ASCII case-insensitive.

## Properties

| Property | Values |
|---|---|
| `display` | `block`, `inline`, `none`, `flex` |
| `flex-direction` | `column`, `row` |
| `color`, `background-color` | `black`, `white` |
| `text-align` | `left`, `center`, `right` |
| `font-weight` | `normal`, `bold` |
| `overflow` | `clip`, `hidden` |
| `font-size` | `8px` through `32px` |
| `border-width` | `0px` through `3px` |
| `padding`, `margin`, `gap` | `0px` through `32px` |
| `width` | `1%` through `100%` |

Property names and values are lowercase ASCII in v1. Negative values,
shorthands with multiple values, `auto`, absolute positioning, transforms,
animation, images, URLs, and every unlisted unit/property/value are rejected.

## Defaults and inheritance

Text color, alignment, font size, and weight inherit. Background, spacing,
border, width, display, direction, and overflow do not. `head`, `meta`, `title`,
and `style` are hidden. `span`, `strong`, `em`, `br`, and text are inline.
Heading sizes are 24, 20, 16, and 12 pixels; headings and `strong` are bold.

One or more `<style>` text nodes in the document are resolved before inline
styles. Nested markup inside `<style>` is rejected. Unsupported or malformed
CSS returns a typed error before layout or framebuffer mutation.
