# Spec: device-runtime

Status: approved automatically by user policy on 2026-08-30.

## Objective

Serialize the shared protocol receiver and physical GDEY075T7 driver. Only a
fully committed, CRC-valid 48,000-byte frame may reach the panel callback.

## Contract

- The caller supplies the sole staging frame and a synchronous `Refresh`
  callback; runtime allocates no second frame.
- `Apply` and `AbortDecode` delegate protocol ownership and fail-closed rules.
- A valid commit queues a frame. A newer valid begin may discard and replace a
  queued but not acquired frame (coalescing).
- `Process(now)` acquires the immutable frame, records refresh start time,
  invokes `Refresh`, releases the lease, and returns `StatusRefreshCompleted`
  or `StatusFailed` plus the local callback error.
- The first refresh is immediately eligible. Later refresh starts are at least
  180 seconds apart, including after a failed callback.
- While `Refresh` runs, the single event-loop owner must not call runtime again.
  No goroutine, timer, logging, transport, or hidden allocation is created.
- Malformed, interrupted, incomplete, bad-offset, bad-ID, and bad-CRC transfers
  never invoke `Refresh`.
- The composed firmware uses TinyGo `scheduler=tasks`. On RP2350, waits of at
  least 10 microseconds use the hardware timer plus ARM `WFE` light sleep,
  rather than a busy loop. Poll intervals are changed only after measuring
  disconnected idle, DTR-attached idle, transfer, and refresh current/latency
  on the physical target.

## Verification

Run `go test ./devruntime`. Tests cover valid refresh, every incomplete/error
path, coalescing, the 180-second boundary, callback failure, lease release, and
steady-state allocations. TinyGo resource checks are repeated in the composed
device build.
