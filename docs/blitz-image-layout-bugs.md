# BLITZ-IMG-001/002: image layout failures

Historical failures, superseded 2026-09-07: the pinned Rust engine was removed,
not patched. Both geometry reproductions now pass as active native Go tests;
the old ignored bodies remain recoverable at `37a1471`. The failure analysis
below is preserved. See [the preservation map](blitz-removal-map.md).

Manager-side bug report and regression entry points, recorded 2026-09-06.
Both failures are unresolved. No upstream issue was created by this project.
This report uses synthetic HTML and pixels only; no private dashboard data.

## Status and scope

- **Fact:** both reproductions already exist in `blitz-probe/tests/images.rs`.
  Their assertions specify correct output, not the observed defective pixels.
- **Applied user decision:** keep image support and skip only these two tests.
  After the initial automated-review rejection, the user clarified: "ні,
  підтримка зображень має бути, але ці тести лише пропустити". The exact
  `CONSTRAINTS.md` exception and reasoned `#[ignore]` attributes are now applied.
- **Fact:** `blitz-probe/src/main.rs` now uses the shared bounded BZR1/BZR2
  decoder. Real Go → PNG decode → BZR2 → Blitz → bitmap delivery works for the
  accepted fixture. Known layout failures remain accepted development debt,
  not repaired behavior or complete CSS-profile acceptance.
- **Guards:** `blitz-probe/tests/image_activation.rs` checks exact image pixels
  through the real executable, rejection without a frame for malformed input,
  and text-only BZR1 compatibility. Alpha, metadata and resource limits stay tested.
- No dependency patch, firmware, wiring, USB operation, panel timing or
  public-input permission changes are part of this deferral.

## Reproduction environment

Linux aarch64, Debian Trixie in the existing project Dev Container. Rust 1.98.1;
current graph locked by `blitz-probe/Cargo.lock`:

| Component | Version |
| --- | --- |
| Blitz DOM, HTML, paint, traits | 0.3.0-beta.2 |
| AnyRender / CPU adapter | 0.13.0 / 0.17.0 |
| Stylo / Taffy | 0.20.0 / 0.14.0 |

Both failures also reproduced on the previous beta.1 dependency set; the beta.2
upgrade did not fix them. Default Blitz features, network clients and SVG are
disabled. There is no browser window, GPU, font requirement or Pico involved.

`Request` has an 8×8 viewport, empty font bytes and one decoded image.
The adapter installs straight RGBA8 through `BaseDocument::load_resource`, with
`Resource::Image`, the matching `epaper-asset:0` URL and `node_id: None`, before
final layout/paint. Output is composited on white, then converted to MSB-first
one-bit rows; 1 means black. Each of the eight rows is one byte.

## BLITZ-IMG-001: intrinsic absolute image collapses

Test: `images_use_intrinsic_size_and_position`.
Source: 2×2 pixels, all opaque black (`[0, 0, 0, 255]` repeated four times).

```html
<!doctype html>
<body style="margin:0">
  <img src="epaper-asset:0" style="position:absolute;left:2px;top:1px">
</body>
```

- **Expected:** intrinsic 2×2 box at (2,1), producing two black pixels on each
  of rows 1 and 2.
- **Measured:** loaded raster is 2×2; layout position is (2,1), but size is 0×0.
  No image pixels appear.
- Expected packed bytes: `[00, 30, 30, 00, 00, 00, 00, 00]` (hex).
- Actual packed bytes: `[00, 00, 00, 00, 00, 00, 00, 00]` (hex).

## BLITZ-IMG-002: explicit CSS box shrinks before object-fit

Test: `image_contain_and_clipping_use_final_layout`.
Source: 2×1 pixels, opaque black followed by opaque white.

```html
<!doctype html>
<body style="margin:0">
  <div style="width:3px;height:3px;overflow:hidden">
    <img src="epaper-asset:0"
         style="display:block;width:4px;height:4px;object-fit:contain;object-position:center">
  </div>
</body>
```

- **Expected:** a 4×4 image element; its 2:1 content fits into 4×2, centered
  vertically at y=1. The parent clips at 3×3 without resizing the image box.
- **Measured:** resolved CSS has width/height 4px and no authored maximum;
  unrounded layout is 3×1.5 (earlier rounded observation: 3×2).
- Expected packed bytes: `[00, c0, c0, 00, 00, 00, 00, 00]` (hex).
- Actual packed bytes: `[80, 80, 00, 00, 00, 00, 00, 00]` (hex).

## Localization and uncertainty

- **Measured:** raw image loading and half-transparent white pixels over both
  black and white backdrops work. The alpha and metadata tests remain active.
- **Measured:** loading before the first resolve or between two resolves did
  not repair either fixture. No missing public resource-loading call was found.
- **Source fact:** beta.2 `src/layout/replaced.rs` uses available space as a
  `max_size` fallback via `.or(available_space.into_options())` and
  `.maybe_min(available_space.into_options())`.
- **Inference:** this clamp can reduce the first measurement, after which
  parent-resolved known dimensions preserve an already-wrong size. This is a
  candidate cause, not a verified fix. Exact first-pass inputs are not traced.
- **Not attempted:** removing that clamp, vendoring Blitz, forcing dimensions
  as an application workaround, or relaxing expected pixels. A future fix must
  also preserve authored min/max sizes, aspect ratio, clipping and alpha.

Related upstream work, checked 2026-09-06:

- [Issue #392](https://github.com/DioxusLabs/blitz/issues/392), open: HTML width
  attributes do not propagate to Taffy. Related, but our explicit-size fixture
  uses CSS and receives the correct computed style; not an established duplicate.
- [Issue #337](https://github.com/DioxusLabs/blitz/issues/337), closed: raster
  images rendered out of proportion. Similar symptoms, not a proven same cause.
- [PR #605](https://github.com/DioxusLabs/blitz/pull/605), merged: preserves
  parent-resolved sizes during flex measurement. Present in beta.2; insufficient
  for these two fixtures. No exact duplicate was found in the bounded search.

## Run and eventual reactivation

Run from `experiments/03-remote-epaper` inside the already-running Dev Container:

```sh
# Normal image target: two passes and exactly two named ignores.
/root/.cargo/bin/cargo test --locked --offline --manifest-path blitz-probe/Cargo.toml --test images -j 2
# Real executable: valid image pixels, malformed requests and BZR1 compatibility.
/root/.cargo/bin/cargo test --locked --offline --manifest-path blitz-probe/Cargo.toml --test image_activation -j 2
```

The applied exception is limited to the two named reproductions above, owned
by the manager-renderer maintainer. It expires at the next renderer dependency
or image-layout implementation change; reevaluate then and obtain approval for
any renewal. No blanket skips, deleted assertions, vendor patch or production
acceptance are authorized. For a future deliberate bug investigation, run:

```sh
/root/.cargo/bin/cargo test --locked --offline --manifest-path blitz-probe/Cargo.toml --test images -j 2 -- --ignored
```

Remove the ignores only after both original assertions pass; a layout fix must
also cover min/max/zero-space, clipping and alpha regressions. Image delivery is
already enabled under the bounded exception. Basic viewport/box fixtures now
pass (see `blitz-viewport-css.md`); next is protected full-refresh timestamp composition.
Broader image-profile, public-input and physical-panel acceptance remain separate.

## Verification

Current activation increment, inside the existing Dev Container:

- Activation test first failed against BZR1-only main; shared-decoder activation
  makes all three real-executable tests pass.
- Full locked offline Rust suite: 16 pass, exactly 2 ignored, 0 failed. Ignored
  assertions were retained and not explicitly rerun. Formatting and all-target
  Clippy with warnings denied pass.
- Go task gate: PASS, 62s; changed coverage 93.1%, total 90.0% against the 75.0%
  ratchet, zero lint issues. Existing file-length debt remains reported.
- `testdata/blitz-image.html` through real `cmd/blitz-preview`: two 128×128
  checkerboards and readable Ukrainian text. The fixture explicitly selects the
  supplied `Go` font; an earlier run without that family omitted its text.
- Read-only review found an incomplete aggregate-limit test packet; adding its
  final RGBA pixel prevents EOF from masking a relaxed limit. Full suite rerun passed.
- Existing TinyGo builds pass, but no firmware was flashed and no panel updated.

## Primary references

- [Blitz beta.2 replaced-element layout](https://docs.rs/crate/blitz-dom/0.3.0-beta.2/source/src/layout/replaced.rs)
- [CSS replaced-element intrinsic width](https://www.w3.org/TR/CSS22/visudet.html#inline-replaced-width)
- [CSS object-fit](https://www.w3.org/TR/css-images-3/#the-object-fit): fitting
  content does not redefine the established element box.
- [Rust test ignore semantics](https://doc.rust-lang.org/reference/attributes/testing.html#the-ignore-attribute):
  ignored tests still compile, and remain explicitly runnable.
- Broader integration and past experiments: `manager-image-assets.md`.
