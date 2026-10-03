# Native CSS declaration core

Historical Rust meaning of "native", superseded 2026-09-07: this opt-in
Stylo/csscheck component was removed, not promoted into the Go renderer.
Current native declarations live in `engine/style`; follow
[SPEC-engine.md](history/remote-epaper/SPEC-engine.md) and [the removal map](blitz-removal-map.md).

Status: implemented as opt-in `csscheck::Validator::native()` on pinned
cssparser 0.37.0 / Stylo 0.20.0. `Default`, public grammar functions and the
current BZR3 worker remain grammar-only. Do not enable this incomplete core for
manager submissions before fallback, font/asset and computed-layout
policies are integrated. Existing image delivery is unchanged.
Native selector admission is now implemented separately; see
[selector and fallback boundaries](native-css-selectors.md).

## Boundary and pipeline

One validator owns a document's existing 32 KiB / 4096-token / 16-depth /
256-declaration budget across inline sources and stylesheets. Each declaration:

1. Passes the existing structure and resource checks.
2. Uses Stylo property/value grammar with fresh state, before cascade merging.
3. Resolves the canonical property ID, including aliases and escaped names.
4. Checks every expanded longhand and rejects deferred `var()` values.
5. Checks allowed tokens and specified numeric values using cssparser again.

No independent tokenizer, property-value parser, renderer patch or dependency
upgrade. Unsupported earlier declarations cannot disappear behind a later valid
override. The parser cursor is restored after policy inspection, including
failure, so source ranges and following parser behavior stay consistent.

## Current core surface

This is an incremental admission policy, not proof that every admitted case
paints correctly. Known Blitz image/positioning defects remain separately waived.
Only Stylo-valid combinations of the following forms are accepted:

| Family | Core values |
| --- | --- |
| `display` | `block`, `inline`, `inline-block`, `none` |
| `position`; physical edges | `static`, `relative`, `absolute`; lengths, percentages, `auto` |
| Width/height and physical min/max | lengths, percentages, `auto`/`none` where grammar allows |
| `box-sizing` | `content-box`, `border-box` |
| Margin/padding and physical sides | lengths/percentages; margin also `auto` |
| Border width/style/color and physical sides | bounded widths or `thin/medium/thick`; `none/hidden/solid/dashed/dotted/double`; colors |
| Border radius and physical corners | length/percentage radii, including elliptical `/` form |
| Overflow and x/y | `visible`, `hidden`, `clip` |
| Color/background-color | named/hex absolute colors, `currentColor`, RGB(A), HSL(A) |
| Font size/weight; line height | lengths/percentages; numeric weight or `normal/bold`; line height also number/`normal` |
| Text align | `left`, `center`, `right`, `start`, `end` |
| White space | `normal`, `nowrap`, `pre`, `pre-wrap` |
| Overflow wrap | `normal`, `break-word`, `anywhere`; canonical `word-wrap` alias works |
| Z-index; opacity | integer/`auto`; number or percentage |
| Object fit/position; aspect ratio | `fill/contain/cover/none/scale-down`; physical edge/center keywords and positions; positive ratio/`auto` |

CSS-wide `initial`, `inherit`, `unset` are admitted, but resolved inherited
values still require the future computed-state gate. `revert*` is not admitted.
Narrow shorthands: margin, padding, overflow, border-width/style/color/radius,
and white-space. Broad resetting `border`, `background`, `font`, and `all` are
not admitted yet: their extra resets need explicit handling, not silent loss.

`font-family`, background URLs and background placement remain resource-policy
work, not permanently removed requirements. URL spelling/escaping does not
authorize a fetch. All CSS custom properties, `var()`, math functions (including
constant `calc()`), transforms, animations and filters are outside this core.
HSL hue is unitless; `180deg` is rejected. Relative/system colors and color-mix
are rejected even if Stylo can parse them. Units are `px`, `%`, `em`, `rem` only.
This pin's standalone grammar already rejects `display:grid` before native policy.

## Numeric limits

These are conservative manager policy choices, not panel limits or CSS standards.
They leave space for offscreen layout while bounding individual input operands:

- Absolute magnitude: 32768 px, 128 em/rem, 1000%.
- Z-index: −32767..32767; opacity: 0..1 or 0..100%.
- Numeric font weight: grammar-valid 1..1000; unitless line height: 0..128.
- Other numeric operands: magnitude at most 32768; ratio components must be
  strictly positive. Normal Stylo sign/unit restrictions still apply.
- Non-finite tokens fail before semantic parsing. Values rejected by the native
  bound are not silently clamped. Percentage opacity uses its own normalized range.

**Not a computed-size guarantee:** `em` inheritance, ratios, percentages and
layout can multiply small operands into large results. Before public use, bound
resolved boxes/coordinates, font metrics, raster allocation and total work, and
reject non-finite output. This component alone is not an untrusted-HTML sandbox.

## Diagnostics and compatibility

Existing codes 1–10 retain their meanings. Local native codes append:

- `UnsupportedProperty` = 11: original property-name span.
- `UnsupportedValue` = 12: full original value span.
- `ValueLimit` = 13: full original value span.

Spans are half-open UTF-8 bytes in decoded CSS, not original HTML. No raw CSS,
font name or URL appears in errors. Structure/grammar failures precede native
policy for that declaration. A value span may include its `!important` suffix.
Codes 11–13 are **not emitted on BZR3**: its default policy is unchanged; future
integration must update and test both sides' diagnostic contract together.
Native selector rejection separately appends local code 14.

## Verification and next integration

Twelve added tests cover positive declarations/aliases, out-of-profile values,
duplicate overrides, all numeric/document boundaries, shorthand operands,
escaped functions/URLs, UTF-8 truncations, source-free spans, unchanged BZR3 and
exact pixels through real Blitz after native validation. No mock renderer.

Run inside the existing Dev Container from `blitz-probe`:

```sh
/root/.cargo/bin/cargo test --locked --offline --test css_native --test css_native_limits --test css_native_render --test checked_wire
```

Remaining: matching each declaration to actual
DOM/fallback ownership; broad shorthand resets; approved fonts/background
assets; computed bounds; public-input isolation. A `<style>` inside a bitmap
subtree is not scoped to it, so source location alone cannot choose fallback.
Neither native admission nor same-engine bitmap rendering repairs Blitz bugs.

Costs: one additional bounded token walk and per-declaration checks only when
native mode is selected. No extra Pico RAM, wire bytes, refresh or SPI activity.
Manager CPU/peak memory/energy are not benchmarked by these tests.

## Primary references

- [Locked Stylo declaration parsing and expansion](https://github.com/servo/stylo/blob/67faaab3ff7aa66780ec1d0f51ca47e177b812d3/style/properties/declaration_block.rs#L1538)
- [Canonical IDs and CSS-wide values](https://github.com/servo/stylo/blob/67faaab3ff7aa66780ec1d0f51ca47e177b812d3/style/properties/mod.rs)
- [cssparser declaration callback](https://docs.rs/cssparser/0.37.0/cssparser/trait.DeclarationParser.html)
- [CSS values, units and computed range checking](https://www.w3.org/TR/css-values-4/)

APIs above were checked against the exact installed source, not an assumed
latest renderer API. The project profile intentionally admits less than CSS.
