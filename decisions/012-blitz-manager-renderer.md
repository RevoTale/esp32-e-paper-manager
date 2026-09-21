# ADR-012: Use Blitz for manager-side HTML/CSS rendering

## Status

Accepted for implementation through an isolated, version-pinned spike.
Production acceptance remains conditional on correctness, security, resource,
and energy measurements.

2026-09-05: `../blitz-probe` now pins the crates and Cargo.lock and passes five
focused CPU pixel-composition tests. It is not the Go worker integration or
production acceptance. Stylo invokes Python during build. On 2026-09-05 the
user approved this upstream build-time generator inside the project's Dev
Container without repeated approval; other Python use remains restricted.

## Date

2026-09-03

## Context

The authoring format must be useful beyond one dashboard. It must accept normal
HTML and modern CSS layout, including block and inline flow, `inline-block`,
absolute positioning, text alignment, the box model, viewport-relative sizing,
backgrounds, and images. The user selected Blitz over the earlier litehtml
candidate because wider modern layout support is worth the added integration
cost.

Blitz is a modular Rust HTML/CSS engine built from Stylo, Taffy, Parley,
html5ever, and AnyRender. Its maintainers explicitly classify it as beta and do
not provide non-Rust bindings.

## Decision

Use exactly pinned Blitz together with `Cargo.lock` as the manager-side
renderer. The initial pin was `0.3.0-beta.1`; the user-approved 2026-09-06
evaluation now pins `0.3.0-beta.2` with AnyRender 0.13.0 / CPU adapter 0.17.0.
Both versions fail the intrinsic/absolute and contain/clipping image fixtures;
the upgrade is **not** image-feature or production acceptance. The initial
decision kept local BZR2 image delivery disabled. **Superseded 2026-09-06:**
the user explicitly enabled images while deferring exactly those two tests;
see the root `CONSTRAINTS.md` exception and `../docs/manager-image-assets.md`.
Keep TinyGo firmware independent: Pico receives only the
bounded resolved display-list or one-bit bitmap protocol.

Expose Blitz through the existing Go `HTMLRenderer` provider. The first
integration is an isolated Rust renderer executable with a small versioned
stdin/stdout contract. Go must not depend on Rust or Blitz types. This process
boundary contains parser crashes and beta API churn; whether the production
manager keeps a warm process or starts it per update is decided only from
measurements.

Build from the narrow modular crates with default features disabled:

- `blitz-dom` for style resolution and layout;
- `blitz-html` for HTML parsing;
- `blitz-paint` for paint commands.

Do not include `blitz-net`, `blitz-shell`, Dioxus, windowing, accessibility,
forms, JavaScript, navigation, or remote resource fetching. Keep SVG support
disabled initially. Assets enter only through the manager's bounded asset
policy.

## Required spike gates

- deterministic 800x480 output and another fake-display size;
- the required HTML/CSS profile, Ukrainian text, local images, clipping, and
  overflow diagnostics;
- bounded malformed and adversarial input, process timeout, output limit, and
  clean crash recovery;
- exact one-bit golden fixtures and stable output across repeated runs;
- cold and warm latency, CPU, peak RSS, binary/container size, wire bytes, and
  manager energy measurements;
- license inventory and a pinned dependency audit;
- no regression in Pico RAM, firmware size, transport, diff, or refresh policy.

Failure of a gate keeps the existing renderer as the compatibility path and
reopens ADR-011's litehtml fallback. It must not weaken the public rendering
contract or silently add a full browser.

## Consequences

- The manager gains a Rust build artifact and a larger dependency graph.
- Modern CSS behavior comes from established browser components instead of a
  project-specific layout implementation.
- Pixel/object diff and Pico partial/full refresh decisions remain independent
  of Blitz.
- The public product contract is the tested e-paper profile, not every behavior
  present in the current beta engine.

## Sources

- https://github.com/DioxusLabs/blitz
- https://github.com/DioxusLabs/blitz/blob/main/Cargo.toml
- https://github.com/DioxusLabs/blitz/blob/main/packages/blitz/Cargo.toml
- https://github.com/DioxusLabs/blitz/blob/main/packages/blitz-html/Cargo.toml
- https://github.com/DioxusLabs/blitz/blob/main/packages/blitz-paint/Cargo.toml
- https://blitz.is/status/css
