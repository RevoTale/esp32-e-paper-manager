# Server grouping policy

Historical renderer binding, superseded 2026-09-07: `renderbatch`/`manager.Screen`
remain active with the native Go engine; `blitzworker`, `blitz-preview` and
`-render-worker` below are checkpoint commands, not current entry points.
Use [engine-authoring-api.md](engine-authoring-api.md) and
[native-preview.md](native-preview.md); see [removal evidence](blitz-removal-map.md).

`renderbatch` is the per-device scheduling core for the server-rendered path.
`manager.Screen` now composes it with a bounded renderer interface, implemented
by `blitzworker.Worker`; `blitz-preview` exercises that actual composition.
Opt-in `epaper-manager -render-worker` now binds a separate scene API and pump
to supervised EPS1 delivery. See `screen-manager-usb.md` for contract/evidence.
It does not use legacy `manager.Store`; legacy delivery remains unchanged.
No new service is deployed.

## Contract

The owner serializes canonical scene changes and assigns strictly increasing,
nonzero `Revision` handles. Submit one handle for one complete immutable target,
including every change from the same HTML update. Never submit independent
subtree patches as unrelated complete targets. Reject stale scene writes before
this boundary; reject stale renderer completions before making them available.

- `Policy{}`: ready immediately. `MaxWait` alone adds no delay.
- `Debounce`: trailing quiet period, reset by each pending update.
- `MaxWait`: cap from the first pending arrival, never reset by later arrivals.
  The earlier deadline wins, even when MaxWait is shorter than Debounce.
- `Submit`: replaces only the pending revision, not the active lease. It cannot
  overwrite canonical scene data because the queue contains no scene data.
- `Wait`: remaining grouping delay; caller can arm one timer. Omitted/zero delay
  does not mean poll continuously while the transport is unavailable.
- `Take`: leases newest ready revision only if no active lease and the caller's
  availability gate permits dispatch. BUSY/offline safety may exceed MaxWait.
- `Release`: only matching active revision releases the lease. It does not
  declare a confirmed frame, retry a failed target or resolve an unknown ACK.

One owner/goroutine calls the methods; concurrent HTTP handlers must serialize
through the coordinator. A released failed/unknown transfer stays unavailable
until the coordinator reconciles device state. Do not clear an active physical
refresh merely because a socket disconnected. New arrivals during an active
lease have their own pending window. Retain latest canonical content outside
the queue for deliberate resync; release does not silently re-enqueue it.

## Resource and clock rules

Constant-size state: policy, three revision handles and three duration values.
No timers, goroutines, bitmap copies or retained update history. Negative policy
durations reject. Use checked duration parsing at external boundaries; this
typed API has no float/JSON conversion, NaN or infinity representation.

Pass `time.Since(serverStart)` consistently. Negative or regressing observations
reject. Elapsed subtraction avoids addition overflow near duration limits.
Source: [Go monotonic clocks](https://pkg.go.dev/time#hdr-Monotonic_Clocks).
Revision wrap requires a fresh coordinator epoch and explicit full resync.

Tests cover omitted settings, MaxWait alone, trailing debounce, continuous
arrivals, shorter MaxWait, BUSY/offline, one active/newest pending, fresh windows,
stale submissions/releases, clock regression and duration overflow. These are
host scheduling tests, not end-to-end scene merging or device power evidence.
Maintenance full-refresh scheduling is separate and must not be starved by
debounce when the coordinator is integrated.

## Verification — 2026-09-05

Focused race tests pass, package statement coverage 100%. Successful queue
cycles allocate zero times after construction in the host allocation test.
Task gate PASS in 17 seconds; worktree changed-line coverage 91.7%, total 89.4%.
Independent scoped review found no required findings; it did not execute tests.
No firmware flashed, physical refresh triggered or network interface enabled.

## Scene coordinator integration — 2026-09-05

`manager.Screen` accepts full UTF-8 scene replacements up to 32 KiB and copies
their input. Submit requires the exact current base revision, preventing a
stale whole-document overwrite. It does not implement DOM edits or merge
independent HTML documents. At this checkpoint API/If-Match mapping was future
composition; its later loopback implementation is in `screen-manager-usb.md`.

RenderNext leases only one ready revision, snapshots its markup and renders
outside the mutex so newer submissions can proceed. If the revision changed
while rendering, it discards the obsolete result and releases its lease; newest
pending work survives. Renderer errors release without confirming any pixels.
There is no automatic render retry: current scene is retained, and the owner
can explicitly resubmit complete content with the current base revision.

Once returned for delivery, a frame is a read-only borrow until Resolve. The
renderer must return independently owned packed 1bpp storage and never mutate
it later. Size/stride and unused row bits are validated. A matching Resolve is
allowed only after successful rendering. Confirmed=true means terminal protocol
success, not visible acceptance; false includes ambiguous failure and clears
the confirmed baseline. Old/mismatched/premature results cannot release another
lease. New pending scenes can coexist with the active delivery.

The coordinator retains revision status, latest markup and one rendering lease;
it does not retain a confirmed pixel baseline for diff replay yet. Profile/time
changes, timestamp composition, device epoch, transport resync, maintenance and
canonical partial-edit semantics remain future integration. Do not expose
Screen directly as a public network handler.

Verification: manager race tests pass (97.1% package statement coverage).
Task gate PASS, 20 seconds; changed-line coverage 92.0%, total 89.5%.
Actual container command `go run ./cmd/blitz-preview
./blitz-probe/target/debug/epaper-blitz-probe ./build/stream/acceptance-4.html
./build/stream/coordinator-preview.png` succeeded. The generated 800×480 PNG
was inspected: the S4 heading, Ukrainian text and lower-right badge are legible.
No Pico update, new network service or firmware change occurred.
Independent scoped review found no concrete blockers; it reviewed ownership,
stale-result rejection and matching resolution, without independently running tests.
