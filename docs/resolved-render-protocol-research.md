# Resolved-render protocol and energy research

Status: recommended design input; wire values and optional operations are not
frozen. Recorded: 2026-09-02. Architecture owner: ADR-008.

Target: Pico 2 W with TinyGo, Waveshare 7.5-inch V2 800x480 black/white panel,
and e-Paper Driver HAT Rev2.3. The Go manager resolves HTML/CSS; Pico receives
bounded drawing work, updates its one-bit framebuffer, and owns panel safety.

## Evidence labels

- **Fact**: stated by a primary specification or current official source.
- **Inference**: a design conclusion from facts that still needs measurement.
- **Research gate**: do not enable in production until software and physical
  acceptance tests pass.

## Recommendation

Use a small, versioned binary display-list protocol carried inside the existing
authenticated and encrypted device-link records. Send one atomic update from a
known base revision to one target revision. The manager keeps the latest desired
render tree and coalesces intermediate changes. Pico validates bounded commands,
rasterizes them, calculates the final pixel dirty regions, and performs at most
one physical panel refresh for the accepted update.

Do not implement VNC/RFB. Borrow its proven ideas: rectangle updates, a mandatory
raw fallback, `CopyRect` for existing pixels, negotiated encodings, explicit
incremental/full resynchronization, and dropping transient states when the
receiver is slower than the producer. RFC 6143 describes these properties but
also marks RRE and Hextile obsolescent; they are design evidence, not formats to
copy.

## Protocol messages

The exact numeric tags remain a specification task.

| Message | Direction | Purpose |
|---|---|---|
| `HELLO` | Pico -> manager | Protocol version, display profile, dimensions, bit depth, supported operations/encodings, hard limits, boot epoch, current revision, and refresh capabilities |
| `UPDATE_BEGIN` | manager -> Pico | Complete or patch update, base and target revision, operation count, dirty bounds, byte count, display/font/asset versions, and displayed timestamp |
| drawing operations | manager -> Pico | Bounded, layout-resolved changes described below |
| `UPDATE_END` | manager -> Pico | Finish the transaction and optionally provide the expected framebuffer identity |
| `ACCEPTED` | Pico -> manager | The complete transaction passed validation and entered the device queue; this does not prove visible pixels changed |
| `REFRESHED` | Pico -> manager | BUSY completed successfully and the target revision became the confirmed device state |
| `UNCHANGED` | Pico -> manager | Raster result equals the current framebuffer; no physical refresh occurred |
| `REJECTED` | Pico -> manager | Typed stage/code plus failing operation index or bounded field name |
| `RESYNC_REQUIRED` | Pico -> manager | Base revision, epoch, profile, cache, or framebuffer state is not safe for a patch; send a complete display list |

Only one update may be in flight per Pico. The manager must calculate the next
patch from the last `REFRESHED` revision, not merely from a sent or `ACCEPTED`
revision. A Pico reboot, unknown revision, changed display profile, or lost asset
cache forces a complete update.

## Commands that reduce bandwidth

Ordered by expected value for a monochrome dashboard:

1. `CLEAR_RECT` / `FILL_RECT`: geometry plus one color replaces a bitmap region.
2. `DRAW_GLYPH_RUN`: the manager sends absolute glyph positions and IDs for a
   negotiated resident bitmap font. Pico does not shape, wrap, or lay out text.
3. `DRAW_ASSET`: draw an immutable icon or bitmap by content-addressed ID after
   `DEFINE_ASSET`; firmware-resident assets require no network upload.
4. `COPY_RECT`: copy already-confirmed framebuffer pixels for scrolling or
   moving an unchanged block. Disable after resynchronization and use an
   overlap-safe copy. Do not source pixels written earlier in the same update.
5. `DRAW_LINE` / `STROKE_RECT`: compact primitives for borders and separators.
6. `BLIT_1BPP_RAW`: mandatory fallback for any bounded region, including a
   manager-decoded `<img>` or explicitly rasterized custom HTML subtree.
7. `BLIT_1BPP_RLE`: optional row-bounded run-length encoding. The manager uses
   it only when the encoded result, including headers, is smaller than raw.

An 800x480 one-bit frame is 48,000 bytes. A raw region costs
`ceil(width / 8) * height` bytes before protocol overhead; a fill, line, glyph,
or cached-asset command can describe the same visual result in tens of bytes.
These are deterministic byte-count savings, not yet measured energy savings.

Exclude PNG, JPEG, SVG, zlib/ZRLE, transferable fonts, arbitrary vector paths,
and general CSS decoders from Pico. The manager may accept documented source
formats, but it resolves them into display-list primitives or one-bit raw
bitmaps. Add Pico-side compression only when a corpus benchmark shows that saved
radio work exceeds decode and code-size cost on the real target.

## Encoding rules

- Fixed binary fields; no JSON, text numbers, or Base64.
- A common operation header with opcode, flags, and length allows bounded
  skipping of negotiated optional operations. Unknown required operations fail.
- Use fixed-width unsigned geometry suitable for the negotiated display bounds.
- Lengths, counts, decoded bytes, clipping, asset IDs, and total work are
  validated before framebuffer mutation.
- Security remains in the existing HMAC/AES-GCM device link. Do not add a CRC or
  hash to every operation: AEAD already authenticates transmitted bytes.
- Revisions provide state ordering. An optional final framebuffer hash is for
  resynchronization and diagnostics, not physical proof that the panel changed.
- Keep records bounded by the existing transport and CYW43439 MTU behavior;
  avoid a tiny record per operation. Batch adjacent operations while retaining
  application-level length limits and no unbounded allocation.
- One update is atomic. The implementation must choose and measure either a
  bounded retained command batch or bounded dirty-region staging before the
  wire format is frozen. Streaming directly into the only framebuffer without
  rollback would expose a partially applied revision after a late error.

## Coalescing and cache policy

- If sanitized source content and display profile are unchanged, the manager
  performs no layout, transmission, or refresh request.
- The manager stores the newest desired render tree per device and only the last
  physically confirmed device revision. While a transfer or panel refresh is in
  flight, newer inputs replace the queued target instead of forming a backlog.
- After `REFRESHED`, calculate one patch from the confirmed tree to the newest
  target. Intermediate dashboard states are intentionally skipped.
- Cache parsed documents, shaped glyph runs, layout results, fonts, and immutable
  assets by content hash and display-profile version on the manager.
- Prefer firmware-resident fonts/common icons, then bounded RAM asset caching.
  Persistent Pico flash caching is a research gate because it adds write wear,
  invalidation, recovery, and storage complexity.
- Manager-decoded images and custom raster subtrees use content identity. Send
  raw bytes only for changed pixels or cache misses; never resend the original
  encoded image format to Pico.
- Merge adjacent dirty bounds on the manager only to reduce command overhead.
  Pico still computes the actual pixel diff and applies local partial/full
  refresh rules.

## Energy policy: manager and network

- **Inference**: stable-ID reconciliation and invalidating only affected layout
  ancestors avoids full HTML layout and raster work for small dashboard changes.
- Maintain at most one render job and one in-flight update per device; cancel or
  replace stale queued work before rendering or encrypting it.
- Reuse the authenticated connection. Repeated WPA3 association, DHCP, and
  device-link setup cost latency and radio activity. Application heartbeats must
  be no more frequent than required for failure detection.
- Prefer one bounded batch over many small writes. Avoid IP fragmentation, but
  do not invent a fixed payload size until actual encrypted-record overhead and
  the pinned network stack are measured.
- Start with the CPU renderer. GPU use is a replaceable manager implementation
  detail and should be enabled only when many-device benchmarks overcome GPU
  startup, transfer, and idle-power costs.
- Record per update: source bytes, display-list bytes, encrypted bytes, operation
  count, coalesced revisions, manager CPU time, Pico render time, SPI bytes,
  refresh mode, BUSY time, and measured current/energy.

## Energy policy: Pico and panel

The product decision is continuous Wi-Fi availability. Do not introduce a
periodic sleep-and-poll device model. Save energy without sacrificing that
availability:

- Initialize Wi-Fi only; do not include or initialize Bluetooth.
- Disable verbose production logging and avoid periodic traffic that does not
  serve liveness or delivery.
- The pinned `cyw43439` v0.1.1 public `Device` API exposes WPA3 join and MTU but
  does not document a public radio power-management setter. Wi-Fi PM1/PM2 is a
  **research gate**, not a promised optimization. If a later supported API is
  adopted, measure connection reliability, latency, packet loss, current, and
  WPA3 behavior before enabling it.
- A rasterized no-op performs no physical panel refresh. Coalesce work while the
  panel BUSY signal is active.
- Keep one 48,000-byte framebuffer and add only bounded scratch storage proven by
  target RAM/stack measurements.
- Use the accepted full-refresh sequence as baseline. Partial refresh remains an
  exact-panel experiment with the current local limits: one-second partial
  minimum, 30-second full minimum, at most ten partials per full, at most 35%
  changed area, and at most four regions. The manager cannot override hard
  firmware safety bounds.
- In full-only operation, finish with the verified Waveshare sequence `0x02`
  (power off), wait for BUSY, then `0x07 0xA5` (deep sleep), and deassert HAT
  power only through the already accepted hardware lifecycle. Deep sleep loses
  controller RAM and requires hardware reset to wake, so the next operation must
  reinitialize and restore the required display data.
- Partial refresh may require controller state to remain available. Compare a
  bounded powered partial session against sleep, reinitialize, and base-image
  reload on every partial request. Until measured and physically accepted,
  retain immediate panel sleep with full-only refresh rather than guessing an
  idle grace period.
- The specification's `0x17 0xA7` automatic power-on, refresh, power-off, and
  deep-sleep sequence may reduce controller idle time. The current official
  Waveshare C driver does not use it; treat it as a separate measured hardware
  experiment, not an initial optimization.
- Waveshare recommends a refresh at least every 24 hours in use to reduce image
  sticking. Whether to schedule a maintenance full refresh when content is
  static is an explicit product/physical-acceptance decision; it must not become
  an undocumented timer.

## What the target e-paper supports

**Facts from the panel specification and current Waveshare driver:**

- 800x480, one-bit black/white update data, on-controller SRAM, bistable visible
  image, and write-only serial SPI data flow.
- Full path: `0x10` old data, `0x13` new data, then display refresh.
- Partial path: `0x91` partial-in, `0x90` partial window, `0x13` new region, then
  refresh. The specification defines `0x92` partial-out, while the current
  official C partial function omits it. Follow the physically accepted sequence;
  do not combine the two sources by assumption.
- Partial horizontal bounds are represented in eight-pixel groups in the
  specification. The current official implementation and an open Waveshare issue
  expose disagreement around partial X addressing. Therefore aligned windows at
  multiple X positions require physical acceptance before production enablement.
- BUSY low means the controller is busy; commands must not be issued until the
  documented ready state, with a bounded diagnostic timeout.
- At 25 C the specification lists 4-8 seconds for a full image update, 8 mA
  typical update current, 0.215 mA typical standby current, and 2 uA typical
  deep-sleep current. These are panel figures, not complete Pico+HAT system
  measurements.

## Required experiments before implementation is called complete

1. Freeze binary field layout, limits, required/optional opcode negotiation,
   transaction staging, cache epoch, and typed error codes with golden vectors.
2. Benchmark representative dashboards: full frame, timestamp-only, task status,
   longer text causing reflow, icon change, scroll, and incompressible bitmap.
3. Compare raw, row RLE, primitives, glyph runs, assets, and `COPY_RECT` by wire
   bytes, manager CPU, Pico CPU/RAM/flash, and total update latency.
4. Test loss/reconnect at every transaction boundary; prove a partially received
   update cannot become the acknowledged framebuffer revision.
5. Physically test partial windows at several aligned X/Y positions, distant
   regions, old-glyph clearing, ten one-second updates, full cleanup, sleep/wake,
   and forced BUSY timeout. Retain full-only operation if any acceptance gate
   fails.
6. Measure complete-system current for connected idle, receiving, rasterizing,
   SPI transfer, full refresh, partial refresh, standby, and panel deep sleep.
   Do not derive system battery life from panel-only datasheet values.
7. Evaluate supported CYW43439 power management separately; reject it if it harms
   continuous reachability or adds reconnection churn.

## Sources

Primary:

- Waveshare 7.5-inch V2 specification:
  https://files.waveshare.com/upload/6/60/7.5inch_e-Paper_V2_Specification.pdf
- Current Waveshare 7.5-inch V2 C driver:
  https://github.com/waveshareteam/e-Paper/blob/master/RaspberryPi_JetsonNano/c/lib/e-Paper/EPD_7in5_V2.c
- Waveshare 7.5-inch HAT manual:
  https://www.waveshare.com/wiki/7.5inch_e-Paper_HAT_Manual
- RFB design reference, RFC 6143:
  https://www.rfc-editor.org/rfc/rfc6143.html
- Pinned CYW43439 Go API:
  https://pkg.go.dev/github.com/soypat/cyw43439@v0.1.1

Secondary, unresolved partial-window evidence:

- Waveshare issue #362:
  https://github.com/waveshareteam/e-Paper/issues/362
