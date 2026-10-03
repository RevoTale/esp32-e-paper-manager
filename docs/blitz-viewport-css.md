# Blitz viewport and basic CSS capability checkpoint

Historical checkpoint, superseded 2026-09-07: the static-wrapper geometry case
now passes in the native engine without a skip. Rust source and old commands
were removed; their original assertions remain at `37a1471`. See
[the removal map](blitz-removal-map.md) and [native preview](native-preview.md).

**Known engine defect, deferred by the user on 2026-09-06:** the eleventh fixture
with a static wrapper between positioned ancestors fails on the same pin.
Only that reproduction is ignored; its expected pixels are unchanged. The ten
checks below remain active. See [`blitz-positioning-gap.md`](blitz-positioning-gap.md)
for explicit reproduction and re-enable conditions. The full profile is not accepted.

Measured 2026-09-06 in the existing Linux arm64 Dev Container, pinned Blitz
0.3.0-beta.2. These are characterization guards for the manager's existing
`render_mono` path, not a new layout engine or complete CSS-profile acceptance.
No production code, dependency, firmware, transport or panel policy changed.

## Pixel fixtures

`blitz-probe/tests/viewport.rs` compares complete MSB-first monochrome frames
against simple expected rectangles. White surroundings and unused row bits are
checked too. CSS pixels map 1:1 to output pixels. Each document explicitly resets
root/body margin and padding and supplies definite 100% width/height.

| Fixture | Verified boundary |
| --- | --- |
| Absolute corner | `right`/`bottom` at 64×48, 296×128, 800×480 and odd 81×49; no fixed-panel assumption |
| Root percentages | Definite 25% position and 50% width/height at two viewport sizes |
| Nested percentages | Uses the smaller positioned parent's dimensions and offset, not the viewport |
| Margin/padding | Normal-flow content offset by both values |
| Min/max | Percentage width below minimum, between limits and above maximum |
| Box sizing | Padding/border outside content-box dimensions versus inside border-box dimensions |
| Relative positioning | Offset paint preserves the following block's flow location |
| Hidden element | `display:none` neither paints nor reserves flow space |
| Overflow | Negative-offset oversized child clipped by `hidden`, visible otherwise |
| Inline alignment | Adjacent inline-block boxes follow left/center/right `text-align` |

The alignment oracle deliberately uses boxes, not environment-dependent fonts.
It does not prove wrapping, glyph shaping or font fallback. Solid overflow
fixtures establish clipping, not every internal positioning detail.

## Reproduce and inspect

From `experiments/03-remote-epaper` inside the already-running Dev Container:

```sh
/root/.cargo/bin/cargo test --locked --offline --manifest-path blitz-probe/Cargo.toml --test viewport -j2
go run ./cmd/blitz-preview ./blitz-probe/target/debug/epaper-blitz-probe ./testdata/blitz-viewport.html ./blitz-probe/target/preview-viewport.png
```

The preview uses the supplied `Go` font. Visual inspection confirms readable
Ukrainian text including Ґ/Є/І/Ї, three alignment cards, borders, percentage-sized
content and an absolute bottom-right label. The label explicitly says local
preview; it is **not** the protected last-full-refresh timestamp feature.
This runs through real Go → BZR1 → Blitz, without USB or device acknowledgement.

## Verification and remaining work

At the original ten-fixture checkpoint, all ten tests passed. Full Rust suite:
26 pass, exactly the two approved image-layout ignores, zero failures. Formatting and
all-target Clippy with warnings denied pass. Independent read-only review found
no Required issue; its two optional precision improvements were incorporated.
These tests passed against existing behavior; no defect repair is claimed.

The previous image path remains enabled; BLITZ-IMG-001/002 and their narrow
test exception are unchanged. CSS grammar/diagnostics, text wrapping and overflow,
font fallback, background assets, full profile conformance, public-input isolation
and physical display acceptance remain open. The subsequent isolated timestamp
component and preview are recorded in `refresh-timestamp.md`; live integration
is still pending. This viewport checkpoint makes no CPU, RAM,
bandwidth or energy improvement claim; only bounded test work was added.

## Semantic sources

- [Containing blocks and percentages](https://www.w3.org/TR/CSS22/visudet.html#containing-block-details)
- [Box model](https://www.w3.org/TR/CSS22/box.html#box-dimensions)
- [Box sizing](https://www.w3.org/TR/css-ui-3/#box-sizing)
- [Relative positioning](https://www.w3.org/TR/CSS22/visuren.html#relative-positioning)
- [Overflow clipping](https://www.w3.org/TR/CSS22/visufx.html#overflow-clipping)
- [Inline content alignment](https://www.w3.org/TR/CSS22/text.html#alignment-prop)
