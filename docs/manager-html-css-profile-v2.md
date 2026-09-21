# Manager HTML/CSS and image profile v2

Superseded as an active profile on 2026-09-06 by `../SPEC-engine.md` and ADR-013.
The new engine accepts inline style only, not selectors/stylesheets. Keep the
history below as regression evidence, not as permission for hidden fallback.

> Active capability contract under ADR-012's implementation and measurement
> gate. Pinned Blitz is the selected engine candidate; this profile, not every
> beta engine behavior, defines what the product promises.

Status: required capability profile; exact grammar, numeric limits, and renderer
choice must be frozen and golden-tested before implementation is accepted.
ADR-008 defines resolved render diffs. ADR-009 defines images and raster fallback.

Known engine limitation, 2026-09-06: a required absolute child inside a static
wrapper is positioned against that wrapper instead of its positioned ancestor.
The user approved deferral without an engine fix: only the exact reproduction
is ignored, with its original assertions retained. Neither grammar validation
nor same-engine bitmap fallback repairs layout. Continue independent work, but
do not claim this positioning requirement met. See `blitz-positioning-gap.md`.

2026-09-06 implementation boundary: bounded PNG decoding, Go HTML asset
preparation/scene budgets and isolated BZR2 raw-asset handoff are implemented.
Blitz beta.1 and the approved beta.2 evaluation both fail intrinsic/absolute
sizing and contain/clipping tests; alpha tests pass. The user chose to enable
images and skip only those two reproductions under the exact root exception.
Real Go → BZR2 → Blitz image rendering now passes a local fixture; this does
not repair the known bugs or accept the full CSS/public-input profile below.
See `manager-image-assets.md` and `blitz-image-layout-bugs.md`.

Basic viewport/box checkpoint: ten real-worker pixel fixtures now cover fixed
viewport sizes, root/nested percentages, positioning, spacing, box sizing,
clipping and inline alignment. A real Go preview confirms readable Ukrainian
text. See `blitz-viewport-css.md` for exact scope; grammar validation, text
overflow/font coverage and the broader profile below are not yet accepted.

Text/cascade checkpoint: nine real-font and four cascade fixtures now verify
wrapping, clipping, inherited text styling and selector precedence. See
`blitz-text-css.md` for precise boundaries. Ellipsis remains unpainted in the
preview; pre-line/break-spaces mapping is incomplete. Raw CSS parser recovery
is not the required typed profile-validation gate. The required surface below
is unchanged; these gaps are not silently accepted approximations.

Grammar integration checkpoint: `csscheck` reuses locked cssparser/Stylo and
checks discovered inline/style sources before Blitz DOM construction. Twenty-one
active CSS tests cover strict syntax, selectors and shared document budgets;
six BZR3/BZE1 wire/process tests include real image output and source-free
rejection before rendering. Manager status records rejected revision/location,
preserves the confirmed frame and accepts the next corrected scene. See
`css-declaration-validation.md` and `blitz-checked-ipc.md`. Native property/value
acceptance, fallback and public-input isolation remain required. A grammar pass
is not a capability promise; raw BZR1/BZR2 probes remain trusted-only.

Native declaration checkpoint: opt-in `Validator::native()` now rejects
out-of-profile properties/values and bounds individual numeric operands, with
canonical aliases and shorthand expansion checked before cascade merging.
This core is not enabled for submissions: actual target/fallback ownership, broad
shorthands, fonts/background assets and computed bounds remain. See
`native-css-declarations.md`; the required product surface below is unchanged.

Native selector checkpoint: the same opt-in validator now admits tag/class/ID,
compound, descendant and comma-list selectors with strict original-token checks.
Eight new tests include real cascade pixels and prove that a stylesheet inside
a bitmap-marked subtree still affects outside nodes. BZR3 remains grammar-only;
fallback target matching/extraction is not implemented by this check. See
`native-css-selectors.md`.

Target ownership checkpoint: isolated `domscope::SceneScope` uses the actual
Blitz DOM and matcher to separate native targets from outermost bitmap owners.
Tests cover parser-repaired trees, mixed targets, nested markers, duplicate IDs
and bounded analysis. See `css-target-ownership.md`.

Declaration binding checkpoint: opt-in `SceneScope::styles()` now discovers
actual-DOM CSS and binds original declaration spans to shared rule/inline target
lists. Fourteen new tests cover source context, exact shared budgets, diagnostics
and unchanged real cascade pixels. It still applies the native core everywhere;
fallback authorization and extraction remain open. See `css-style-bindings.md`.

Source identity checkpoint: Go now always emits BZR4 with element ordinals,
including an explicit empty CSS manifest. After grammar validation, the worker
verifies exact sources in the same DOM it paints. Fourteen added Rust tests,
Go serialization fixtures and real image/HTML previews pass. Native policy is
still opt-in; this does not establish full DOM equivalence or public-input
isolation. See `css-source-identity.md` and `blitz-checked-ipc.md`.

Static HTML admission checkpoint: Go now rejects scripts, forms, embedded/media
content, interactive containers, event/navigation attributes and meta pragmas
before asset decoding or worker launch, regardless of bitmap markers. Five
tests plus review cover the boundary; see `static-html-admission.md`. This is
not the full element/attribute/CSS allowlist or public-input sandbox.

## Two rendering paths

### Native resolved path

Use this path when every element, selector, and property belongs to the profile.
The manager produces stable-ID text, geometry, asset, and bitmap objects so a
small semantic change can remain a small display-list patch.

### Explicit raster fallback

`data-epaper-render="bitmap"` marks a subtree that the manager must render into
its final clipped one-bit pixels. The subtree remains one stable render object;
its changed pixel rectangles are sent with `BLIT_1BPP_RAW`. Unsupported syntax
without this marker is rejected with the element/property/value and source
location. The renderer never silently switches paths.

The fallback may cover one widget or the entire body. Larger areas preserve
custom output but cost more manager CPU, network bytes, Pico writes, and panel
dirty area. It therefore remains an explicit author choice.

The fallback is a paint mode of the bounded manager renderer, not permission to
launch a general browser. Chromium/WebKit and JavaScript are excluded from the
default runtime. See `lightweight-renderer-research.md` for evaluated
alternatives and the benchmark required before adding another engine.

## Required HTML surface

The manager profile includes the existing structural and text elements plus:

```text
img figure figcaption
```

`img` supports `src`, `alt`, `width`, `height`, `id`, `class`, `style`, and the
project fallback marker. Missing or failed images render documented `alt` text
or return a typed error according to the submitted update policy.

Scripts, event-handler attributes, forms, navigation side effects, plugins,
audio, video, animated image playback, and dynamic DOM mutation remain
unsupported. An animated source, if an accepted codec can decode it, uses only
its first frame.

## Required lightweight CSS surface

The native path intentionally implements a small dashboard-oriented subset. It
does not aim for browser compatibility:

| Group | Required properties or behavior |
|---|---|
| Cascade | element, class, ID, compound and descendant matching; inheritance; inline style |
| Flow | `display: block/inline/inline-block/none`; normal block and inline flow |
| Box | `box-sizing`, `width`, `height`, `min-*`, `max-*`, `margin`, `padding`, `border`, `border-radius`, `overflow` |
| Text | server-selected `font-family`, `font-size`, `font-weight`, `line-height`, `color`, `text-align`, `white-space`, `overflow-wrap`, `text-overflow` |
| Positioning | `position: static/relative/absolute`, `top`, `right`, `bottom`, `left`, and bounded `z-index` |
| Background | `background-color`, one `background-image: url(...)`, `background-position`, `background-size`, and `background-repeat` |
| Images | intrinsic dimensions, `aspect-ratio`, `object-fit`, `object-position`, clipping, and alpha composition |
| Units | bounded `px`, `%`, `em`, and `rem` resolved against the provisioned display profile |

This table is a required product surface, not a claim that every CSS value or
browser edge case is supported. Flexbox, Grid, tables as layout algorithms,
custom properties, gradients, shadows, filters, and transforms are outside the
required profile. Each supported property receives a documented value
grammar, default, inheritance rule, limit, unsupported-value error, and golden
fixture. Media queries, print pagination, transitions, animations, blend modes,
and arbitrary transforms require an explicit profile extension or raster
fallback.

## Image pipeline

1. Resolve `src` through the manager's asset policy.
2. Bound encoded bytes before reading and decoded dimensions/pixels before full
   allocation; verify actual MIME/codec rather than trusting a filename/header.
3. Decode orientation and alpha, calculate CSS size and clipping, then resample
   once to the final display-space dimensions.
4. Composite against the resolved background and convert to one-bit using the
   display profile's deterministic threshold/dither policy.
5. Emit row-major one-bit pixels with explicit width, height, stride, polarity,
   clipping rectangle, and content identity as `BLIT_1BPP_RAW`; use a negotiated
   immutable asset ID only when caching is confirmed.
6. Pico validates every bound and byte count before framebuffer mutation. It
   does not parse image metadata or allocate from source dimensions.

The manager API should prefer uploaded or content-addressed assets. Data images
may be allowed within separate encoded/decoded limits. Remote HTTP(S) images are
off by default because unrestricted fetching creates SSRF and availability
risks; enabling them requires the ADR-009 policy.

## Diff and energy behavior

- An unchanged content hash emits no bitmap bytes.
- Moving an unchanged cached image may use `DRAW_ASSET` or `COPY_RECT` when the
  confirmed revision makes that safe.
- A changed raster fallback sends only its final dirty rectangles, not its HTML,
  CSS, source image, or unchanged pixels.
- The manager may compare raw and row-RLE representations, but raw remains the
  mandatory Pico fallback and RLE is used only when the complete wire result is
  smaller.
- The Pico still decides unchanged/partial/full refresh from the final pixel
  diff. HTML element boundaries cannot force panel refresh mode.

## Acceptance tests

- Native block cards, aligned/wrapping text, margins, padding, backgrounds,
  positioned images, overflow, and absolute badges.
- PNG/JPEG-class raster fixtures once their codecs are selected; corrupt,
  truncated, oversized, extreme-ratio, transparent, and mislabeled inputs.
- `contain`, `cover`, `fill`, `none`, `scale-down`, and object positioning.
- Custom subtree fallback with unsupported CSS; the same CSS outside fallback
  must return a stable typed error.
- Exact golden one-bit output at multiple display sizes and renderer versions.
- Stable-ID image replacement, movement, removal, cache loss, reconnect, and
  complete resynchronization.
- Wire-byte, CPU, RAM, latency, and energy comparison against a 48,000-byte full
  frame and the equivalent native display-list operations.

## Standards used for semantics

- HTML: https://html.spec.whatwg.org/multipage/
- CSS Display: https://www.w3.org/TR/css-display-3/
- CSS Images: https://www.w3.org/TR/css-images-3/
- CSS Overflow: https://www.w3.org/TR/css-overflow-3/
