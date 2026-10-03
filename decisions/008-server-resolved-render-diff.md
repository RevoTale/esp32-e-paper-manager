# ADR-008: Send resolved render diffs to Pico

## Status

Accepted direction. The binary wire format and resource limits remain to be
specified and tested before implementation.

## Date

2026-09-02

## Context

The original v2 direction sent complete HTML to Pico 2 W. Pico parsed HTML and
CSS, calculated layout, rasterized a complete frame, compared it with the
previous frame, and selected a safe e-paper refresh. That kept both transports
symmetrical, but placed parsing, layout, persistent document state, and diff
work on the constrained device.

The intended model is closer to a server-rendered UI protocol: the Go manager
owns HTML/CSS adaptation, layout, and reconciliation. Pico receives a bounded,
typed representation of only the resolved visual changes. This reduces radio
traffic and embedded complexity without reducing the firmware's authority over
panel safety.

## Decision

For the manager-to-Pico path, use a versioned resolved-render protocol:

1. The manager accepts bounded HTML/CSS and converts it to a typed render tree.
2. The manager calculates layout for the provisioned display profile.
3. The initial update sends a complete resolved display list.
4. Later updates send only changed or removed objects plus every dirty region
   affected by their old and new bounds.
5. Render objects use stable IDs, absolute geometry, clipping, z-order, and
   allowlisted operations such as clearing or filling a rectangle, drawing a
   glyph run, line, or bounded bitmap.
6. Pico validates the message, rasterizes the supplied resolved operations into
   its monochrome framebuffer, and selects unchanged, partial, fast-full, or
   full refresh through its local safety policy.

Pico does not parse manager-supplied HTML, resolve CSS, wrap text, or calculate
container layout on this path. A semantic change is not automatically a safe
display boundary: when it changes wrapping, geometry, overlap, or siblings,
the manager includes all affected render objects and both the old and new
dirty bounds.

The protocol is transport-independent and may be carried over the authenticated
device link or USB. Direct HTML-over-USB remains an explicit compatibility and
recovery capability until its final role is decided; it must not silently
produce different visible semantics from the manager pipeline.

ADR-009 extends the manager input with `<img>` and an explicit server-raster
fallback. Both end as the same bounded bitmap operation; neither adds image or
general HTML/CSS decoding to Pico.

## Synchronization and safety

- Messages identify a base revision and target revision. Pico acknowledges a
  target only after a successful physical refresh.
- A reconnect may continue with a diff when both peers agree on the active
  revision. Unknown, rejected, or inconsistent state requires a complete
  display list.
- Revision metadata coordinates delivery; the manager remains the source of
  truth for HTML, layout, render-tree state, and object diff.
- The firmware validates bounds, operation count, decoded bitmap bytes, font
  and display-profile versions, dirty-region count, and total work before
  mutating the framebuffer.
- The manager cannot force partial refresh, disable periodic full cleanup, or
  override firmware timing, area, fragmentation, and consecutive-partial
  limits.
- Wi-Fi remains continuously managed for responsive updates. This decision
  does not introduce periodic Pico wake-and-poll behavior.

## Why not send only changed HTML nodes?

A changed node can alter text wrapping, parent dimensions, sibling positions,
clipping, and overlap. Sending only semantic values would require Pico to own
the layout engine again. The manager therefore sends layout-resolved drawing
objects and the complete invalidation consequences of the change.

## Alternatives considered

### Complete HTML on every update

Simple stateless input and already represented by the current v2 prototype, but
it repeats parsing, layout, and reconciliation work on Pico and transmits more
data than needed for small dashboard changes.

### Semantic object diff with layout on Pico

Closest to a virtual DOM, but it preserves the most complex and memory-sensitive
part of browser-like behavior on the embedded device. Rejected for the manager
path.

### Server-generated pixel rectangles

Makes Pico smallest and produces exact pixels, but transfers more data and
couples every update to server rasterization. Keep as a bounded bitmap operation
and recovery fallback, not the only update representation.

## Consequences

- The Go manager needs deterministic HTML adaptation, layout, reconciliation,
  per-device display profiles, and confirmed revision state.
- Pico needs a small display-list decoder and rasterizer, not a network-facing
  HTML/CSS engine.
- Fonts and raster assets require explicit versioning or resolved glyph data so
  server geometry and Pico pixels cannot silently diverge.
- Object diffs are usually much smaller than complete 48,000-byte monochrome
  frames, while the final pixel diff and refresh decision remain device-local.
- GPU acceleration remains an optional manager implementation detail behind a
  renderer interface; it cannot change protocol semantics or monochrome output.
- Tests must cover moved and removed objects, longer text, overlap, old-pixel
  clearing, reconnect ambiguity, full resynchronization, malformed operations,
  and firmware-enforced refresh limits.

## Open protocol work

The researched candidate protocol, bandwidth operations, energy policy, and
physical acceptance gates are recorded in
`../docs/resolved-render-protocol-research.md`. It guides the next specification
step but does not freeze wire values or enable experimental panel modes.

- Freeze the complete-display-list, patch, acknowledgement, resynchronization,
  error, and capability-negotiation messages.
- Choose bounded binary encoding without Base64 bitmap payloads.
- Define font/glyph strategy, renderer and display-profile versioning, resource
  limits, idempotency, and USB behavior.
- Measure manager render cost, radio bytes, Pico RAM/flash/CPU, SPI time, panel
  energy, and visible partial-refresh behavior on the physical hardware.
