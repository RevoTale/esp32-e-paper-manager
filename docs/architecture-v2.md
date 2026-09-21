# V2 architecture

## Data flow

Manager path:

`bounded HTML/CSS + assets -> pinned Blitz candidate -> resolved render tree/bitmap ->
stable-ID diff -> encrypted display list -> Pico validation/rasterization ->
pixel diff -> safe panel refresh -> panel power policy`

An `<img>` or explicit custom raster subtree is decoded/rendered on the manager,
converted to deterministic one-bit display-space pixels, and transported as a
bounded raw bitmap operation. Pico never receives PNG, JPEG, SVG, general CSS,
or executable browser content. See ADR-008, ADR-009, ADR-011, ADR-012, and
`manager-html-css-profile-v2.md`.

Direct USB compatibility path:

`HTML <=32 KiB -> older bounded on-device profile -> 1-bit framebuffer -> safe
refresh`

The host USB tool may instead run the manager renderer and send the same display
list as Wi-Fi; this is the preferred route for feature parity.

The display-list contract enters through USB CDC or the outbound device link.
USB is the default and has priority. Network rendering can be cancelled before
refresh; an active physical refresh is never interrupted.

The firmware owns one 48,000-byte framebuffer. Display-list parsing,
rasterization, bitmap input, and dirty-region workspaces are statically bounded.
The timestamp is manager/host-created and rendered in a reserved bottom-right
rectangle that page content cannot use.

## Modules

- `document`, `html`, `css`, `layout`, `raster`, `render`: bounded compatibility
  components reused by the direct-USB recovery path.
- `manager-html-renderer`: provider boundary for the isolated pinned Blitz
  renderer; it is accepted only for the ADR-012 spike until its gates pass.
- `manager-renderer`, `render-diff`: deterministic manager layout, stable render
  objects, invalidation, and complete/patch generation.
- `manager-assets`, `manager-raster-fallback`: bounded image processing and
  explicit custom-subtree one-bit rasterization.
- `display-list`, `device-rasterizer`: transport-neutral resolved operations and
  Pico-side validation/rasterization.
- `display`, `devicebootstrap`, `htmlruntime`: reusable display contract and
  refresh scheduler.
- `panel`: exact Waveshare 7.5-inch V2 full-refresh/deep-sleep adapter.
- `deviceusb`: USB HTML/provisioning record multiplexer.
- `provision`: USB-only atomic credential journal.
- `manager`, `managerclient`: trusted HTTPS control plane.
- `securetransport`, `devicelink`: authenticated encrypted Pico-initiated link.
- `devicearbiter`: USB/network ownership and terminal-result handoff.
- `network`: endpoint policy, entropy adapter, reconnect/backoff policy.

Bluetooth is deliberately absent. Identical pixels are never refreshed.
Production retains the accepted full-refresh baseline. Partial refresh uses the
separately documented one-second/30-second bounded proposal only after exact
panel physical acceptance; the manager cannot force it.

## Delivery status semantics

`accepted` means the complete display-list update was authenticated, validated,
and queued. It does not mean the panel changed. `refreshed`, `unchanged`, or
`rejected` is terminal. Only `refreshed` advances the manager's physically
confirmed revision. A reboot, cache-epoch change, or unknown revision requires a
complete display list rather than an incremental patch.
