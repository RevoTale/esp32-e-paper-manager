# Spec: usb-transport

Status: approved automatically by user policy on 2026-08-30.

## Objective

Carry the approved frame protocol over Pico 2 W USB CDC without assuming USB
packet boundaries. USB is the default and later has priority over Wi-Fi.

## Contract

- One non-zero runtime-local session ID is assigned on every DTR rising edge.
- On attach, send one `Hello`; then decode an arbitrary fragmented byte stream.
- Consume a decoded record immediately because its payload aliases decoder
  storage; serialize one complete ACK/Error/Status before another reply.
- `writeAll` handles positive short writes, zero-progress writes, errors, and
  DTR loss. Device code checks DTR because TinyGo USB CDC otherwise reports a
  successful write while dropping output when DTR is false.
- A typed decoder failure calls `AbortDecode`; USB retains bounded decoder
  resynchronization. A local invariant error closes the session.
- Detach resets only an incomplete owned transfer. A committed queued frame is
  retained for display.
- Fixed buffers only: decoder 284 bytes, reply 284 bytes, device RX 256 bytes;
  no goroutine, log stream, or text mixed with protocol bytes.

## Verification

Run `go test ./usbtransport`, then TinyGo-build `cmd/device`. Tests split every
record across 64-byte boundaries, exercise short/zero/error writes, corruption,
detach/re-attach, and steady-state allocations.
