# ADR-010: Use viewport-first markup instead of general CSS (rejected)

## Status

Rejected after the authoring requirement was clarified. ADR-011 supersedes this
direction with normal HTML and a useful CSS subset. The proposed scene language
is retained only as design history.

## Date

2026-09-03

## Context

The product needs predictable dashboard placement, text, backgrounds, images,
and absolute overlays. It does not need browser compatibility. Even bounded CSS
cascade plus general Flexbox/Grid adds parser, layout, testing, and compatibility
cost that does not improve the e-paper result.

## Decision

The manager accepts a versioned HTML-like scene document defined by
`docs/viewport-layout-profile-v1.md`. The root is an explicit viewport. Layout
and paint are expressed as typed element attributes, not CSS declarations.

The required primitives are `screen`, `stack`, `box`, `text`, `image`, and
`spacer`. A stack provides deterministic row or column flow. Explicit geometry,
edge constraints, and anchors provide absolute placement. Every renderable
element has an optional stable `id` for manager-side diffs.

CSS stylesheets, `<style>`, the `style` attribute, JavaScript, and browser DOM
behavior are outside the contract. Existing HTML is supported only through an
explicit adapter that maps known input into this scene model or returns typed
unsupported-input diagnostics.

## Consequences

- The manager needs a small layout engine, not a browser renderer.
- Viewport adaptation and overflow behavior become explicit and testable.
- Authors use a narrow declarative format rather than arbitrary web pages.
- Pico still receives resolved display-list operations, never markup.
- Unknown elements or attributes are errors rather than ignored browser quirks.
