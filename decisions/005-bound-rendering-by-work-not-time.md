# ADR-005: Bound rendering by work, with a watchdog for stalls

## Status

Accepted.

## Date

2026-09-02.

## Context

On-device HTML parsing and rendering must terminate predictably without
rejecting valid content merely because one execution was slower. E-paper panel
refresh is independently slow and must not be confused with software rendering.

## Decision

Limit encoded HTML to 32 KiB. Add explicit byte, node, depth, text, asset, and
parser/layout work limits before implementation; those deterministic limits are
the primary denial-of-service and termination controls.

Set a 5-second Pico 2 W target from complete input to validated framebuffer.
Set a separate 15-second watchdog to recover only from stalled or defective
execution. The watchdog must report the exact stage and counters observed; it
must not select a fallback layout or treat elapsed time as document semantics.

Exclude panel power-up, SPI transfer, physical refresh, and BUSY waiting from
the 5-second renderer target. Measure and diagnose those stages separately.

## Alternatives considered

### Two-second hard rendering timeout

Rejected. It was too restrictive as an initial bound and incorrectly made wall
time part of content validity.

### No watchdog

Rejected. Deterministic budgets should prevent normal runaway work, but a
defect must not leave the device permanently unresponsive.

### Include panel refresh in rendering latency

Rejected. It hides whether latency comes from parsing/layout or the physical
e-paper lifecycle and makes performance regressions harder to diagnose.

## Consequences

- Tests must cover exact boundary values and prove no display mutation on
  oversized or work-budget failures.
- Diagnostics must distinguish target miss, work-budget rejection, watchdog,
  and panel-stage failure.
- Limits may change only from repeatable physical resource and latency evidence.
