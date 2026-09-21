# Rendering decisions and remaining implementation evidence

Status: design, not implemented. This amendment supersedes conflicting timestamp
and USB authoring wording in earlier v2 documents. The 2026-09-05 user
clarifications below also supersede mandatory native Pico IPv6 and implicit
batching delays. Exact-panel partial-refresh safety remains open.

## Server batching (user clarification, 2026-09-05)

The server owns update cadence and optional grouping of independent updates.
All visual changes from one HTML update always form one atomic transaction,
regardless of whether batching is configured.

Both per-device duration settings are optional, with no implicit positive default:

- `debounce_interval`: wait after the latest independent update. Each arriving
  update resets this quiet-period deadline.
- `max_wait`: cap grouping time from the first pending update. New arrivals
  never reset this deadline.

Omitted or zero `debounce_interval` means no intentional grouping delay: submit
as soon as the device is available. `max_wait` alone does not introduce a delay.
With debounce configured and max_wait omitted, ordinary trailing debounce
applies; continuous input can postpone submission until a quiet period. Setting
max_wait bounds that delay. Reject negative, non-finite or overflowing durations.
If both deadlines exist, the earlier one makes the batch ready, including when
max_wait is shorter than debounce_interval. Use a monotonic server clock.

Device BUSY, an in-flight transaction, transport availability and hardware
safety can delay actual delivery beyond max_wait; it bounds grouping, not panel
completion. Retain one newest pending target, preserve unrelated changes, and
compute the next patch from the last confirmed revision. Start a new batching
window for new updates after dispatch. Never split one HTML change into several
physical refreshes solely because it produced several drawing operations.

Pico enforces safe panel sequencing and reports deferral/rejection, but adds no
independent configurable debounce. The maintenance full-refresh interval is a
separate policy; max_wait does not postpone already-due maintenance indefinitely.
Acceptance tests cover omitted/zero settings, each setting alone, simultaneous
and continuous arrivals, earliest deadline, BUSY, reconnect and atomic HTML updates.

Implementation checkpoint, 2026-09-05: `renderbatch` now implements the isolated
revision-handle queue and grouping/lease policy, with 100% package coverage and
race tests. See `server-batching.md`. Canonical scene editing, render generations,
maintenance scheduling, actual transport integration and public configuration
remain coordinator work; the existing manager behavior has not changed yet.

## Blitz findings

The live official CSS status table reports inline/block/inline-block, flex/grid,
text-align, padding/margin, background images/position/size, and object-fit/position.
It reports partial tables, missing fixed/static positioning and text-overflow;
absolute elements are described as relative to their immediate parent.
These are live documentation findings, not verification of our pinned beta.
Test nested containing blocks, viewport units/media queries, Ukrainian shaping,
inline baselines, clipping, and image positioning on the exact locked build.
README goals must never be treated as implemented feature guarantees.

Blitz resolves layout on the manager. An adapter translates its paint output
into our screen commands. Unsupported paint primitives become final 1-bit
regions. No assumption is made that AnyRender supplies stable DOM IDs or a
ready-to-send incremental Pico protocol. Full layout recomputation may initially
be necessary; bounded wire diffs do not prove incremental layout execution.
USB host tooling uses the same manager renderer. Legacy Pico HTML remains an
explicit recovery capability with its own narrower profile.

## Screen protocol design

Later clarification, 2026-09-05: the user selected controller-RAM streaming for
investigation instead of automatically enlarging the receive buffer. Read
`controller-ram-streaming.md` before implementing this section. Complete-batch
retention and mandatory device mask/cache operations below are earlier design
assumptions under reconsideration, not requirements to implement blindly.
Visible-update atomicity, authenticated delivery and safe refresh remain required;
staging-memory atomicity needs the explicitly documented candidate contract.

Conflict analysis and module/test boundaries are recorded in
`render-conflicts-and-module-boundaries.md`. One target snapshot precedes patch
encoding; renderer outputs from different scene generations must not be merged.
`renderdiff` now implements the isolated final-frame damage reference, not the
screen codec or Blitz integration.

Choose a small binary command stream, using RFB rectangle/cache ideas rather
than adopting VNC. CBOR and Protocol Buffers describe values but do not solve
render invalidation; a closed opcode stream avoids field names and generic
decoder machinery. This is a design choice, not a proven global byte minimum.

Version 1 records use little-endian fixed headers: version u8, kind u8, flags
u16, payload length u32. UPDATE payload begins with boot epoch u64, base revision
u32, target revision u32, command count u16. Revision wrap requires a new epoch
and full synchronization. Operation header: opcode u8 plus bounded unsigned
LEB128 payload length (at most 3 bytes, canonical shortest form). Coordinates
and dimensions are u16, in final clipped device pixels. No Base64.

Mandatory operations: FILL_RECT, BLIT_RAW, DEFINE_MASK, DRAW_MASK_RUN.
FILL_RECT is x/y/w/h plus u8 color; BLIT_RAW is x/y/w/h plus row-major MSB-first
bits, 1=black, zero padding. Images are opaque final composited pixels.
Masks are bounded 1-bit glyph/icon coverage, with dimensions and a session u16
handle; runs carry handles and absolute positions plus color. The server shapes
and rasterizes fonts; Pico neither loads font files nor computes metrics.
Cache epoch and explicit successful definitions prevent stale handle reuse.
Cache misses reject the update without drawing; server substitutes raw regions.

Optional encodings: row-bounded RLE (positive alternating bit runs, canonical
LEB128 counts, exact row width) and COPY_RECT. Negotiate before use. COPY_RECT
reads only the confirmed base frame and requires staged source data for overlap
or dependencies; omit it from the first implementation if staging costs exceed
savings. For each region compare complete encoded costs, including cache misses,
headers and record overhead, and choose the cheapest supported representation.

Server retains scene IDs and restores all affected backgrounds/overlaps; Pico
retains pixels and a bounded mask cache, not a DOM or scene graph. Dirty patches
include erased old bounds and final composited replacements. Run raw-only golden
output alongside optimized commands to prove identical pixels.

HELLO negotiates dimensions, protocol/profile, epoch, cache capacity, maximum
record/transaction bytes and operations. UPDATE carries full/patch intent;
RESULT distinguishes APPLIED_UNCHANGED, REFRESHED_PARTIAL, REFRESHED_FULL,
REJECTED and RESYNC_REQUIRED. Both unchanged and refreshed success advance the
logical revision; last-full time advances only after successful full refresh.
One in-flight transaction, one newest pending target; reconnect ambiguity forces
resync. Refresh failure invalidates the framebuffer baseline until resync.

Retain the complete bounded command batch, authenticate and validate every
operation and asset dependency before framebuffer mutation. No second full
frame is assumed. Large scenes fall back to a bounded full raw frame transaction
instead of an unbounded command list. Exact staging/cache limits require Pico
RAM measurements before freezing constants. TCP/device AEAD and USB framing
carry the same command payload; do not invent new cryptography here.

## Timestamp and full refresh

User decision: default full-refresh interval is 600 seconds, configurable.
User clarification, 2026-09-05: the server owns this schedule. If the server is
unavailable, Pico waits; it does not autonomously refresh cached content every
600 seconds. The visible image and timestamp remain unchanged. Normal reconnect
handling and explicit USB updates remain available. An already accepted refresh
may finish safely after a disconnect. On reconnection, synchronize state and
resume server scheduling without replaying missed maintenance intervals.
Any earlier successful full refresh resets the due time. Partial refresh keeps
the displayed timestamp unchanged. No backlog or catch-up refresh storm.
Due maintenance refresh may reuse cached content; it need not retransmit a
complete image. Preserve the reserved bottom-right rectangle and provisioned
timezone. The display label denotes the start time of the last successful full
cycle; completion time is returned in status. Exact completion cannot be painted
in that same refresh because it is known only afterwards.
Failed refresh never advances the confirmed timestamp. Offline trusted-time
loss leaves the previous label and reports time unavailable; do not invent UTC.
The 600-second cadence is a requested policy, pending exact-panel acceptance.

## IPv6

User clarification, 2026-09-05: native Pico IPv6 is optional future work and
does not block the first release. Use Pico IPv4 to reach the central manager.
Manager IPv6 access can be provided independently by the host/gateway. This
supersedes earlier requirements to finish native dual-stack for first release;
it does not weaken WPA3-only or application-link authentication/encryption.

Current project pins lneto v0.1.0 and cyw43439 v0.1.1. Its network policy explicitly
rejects IPv6-only endpoints with ErrIPv6Unavailable. Upstream IPv6 frame/ICMP
types do not establish a complete usable host stack.
Keep native dual-stack as a separate implementation gate: IPv6 Ethernet receive
path, ICMPv6, neighbor discovery, router advertisements/lifetimes, SLAAC and DAD,
DNS AAAA, TCP checksums/routing, PMTU, reconnect and prefix-change tests.
Evaluate maintained upstream support before implementing missing mechanisms.
Manager dual-stack with Pico IPv4 is useful deployment compatibility but must
never be called native Pico IPv6. No networking dependency migration is made by
this design review.

## Sources

- https://blitz.is/status/css
- https://github.com/DioxusLabs/blitz
- https://www.rfc-editor.org/rfc/rfc6143.html
- https://www.rfc-editor.org/rfc/rfc8949.html
- https://github.com/soypat/lneto/tree/main/ipv6
- https://www.rfc-editor.org/rfc/rfc4861.html
- https://www.rfc-editor.org/rfc/rfc4862.html
