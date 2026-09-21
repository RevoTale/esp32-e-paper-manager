# ADR-006: Borrow one framebuffer across synchronous display refresh

## Status

Accepted.

## Date

2026-09-02.

## Context

The renderer and physical display adapter need a replaceable monochrome
contract that also fits Pico 2 W memory. A hidden copy of the 800x480 1-bit
frame would cost another 48,000 bytes and violate the accepted one-frame
budget. Different panels may also require different sizes and row alignment.

## Decision

Represent a frame as a validated mutable view over caller-owned packed bytes.
The caller retains ownership. `Device.Refresh` borrows those bytes only for the
duration of the synchronous call, may not mutate them, and may not retain them
after returning. Copies of `Frame` remain views over the same storage.

Use `0=white`, `1=black`, most-significant-bit first. Capabilities provide the
logical size, monochrome model, supported refresh modes, exact aligned stride,
and whether a successful refresh already leaves the panel asleep. Every device
must support full refresh; partial refresh remains optional and unaccepted for
the current Waveshare adapter. `Sleep` is idempotent.

Keep the shared interface narrow because rendering and runtime need the same
stable boundary. Concrete panel construction remains in the panel package. The
reusable fake records only frame metadata and an allocation-free digest, never
a second frame.

## Alternatives considered

### Let each panel accept `[]byte`

Rejected. It hard-codes dimensions and ownership assumptions outside the
adapter and cannot validate a second display size independently.

### Copy every frame into the display implementation

Rejected. It makes ownership simple but doubles the dominant RAM allocation.

### Expose asynchronous refresh

Rejected. It would require a lease or second buffer while physical refresh is
active. The current runtime and panel driver are synchronous, so the added
state and memory have no accepted use.

## Consequences

- Renderers can target any validated monochrome size without importing the
  Waveshare driver.
- A caller must not modify the backing bytes until `Refresh` returns.
- An adapter must validate capabilities, mode, size, and exact stride before
  touching hardware.
- Future asynchronous or partial-refresh support requires a new accepted
  ownership/lifecycle decision and physical evidence.

## Sources

- Go interfaces and consumer-owned boundaries:
  https://go.dev/wiki/CodeReviewComments#interfaces
- TinyGo 0.41 standard-library support reference:
  https://tinygo.org/docs/reference/lang-support/stdlib/
