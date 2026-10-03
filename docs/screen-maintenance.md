# Confirmed pixels, full-cycle stamps, and maintenance

This manager-only contract implements the C3/C4 scheduling boundary in
[SPEC-screen-session](history/remote-epaper/SPEC-screen-session.md). It does not establish optical
panel acceptance, change the refresh waveform, or enable partial refreshes.

## Construction and ownership

`NewScreenWithOptions(renderer, size, policy, ScreenOptions{...})` enables this
path. `Zone` is required and comes from authenticated enrollment. A zero
`MaintenanceInterval` selects 600 seconds; a positive duration overrides it;
negative durations reject. `Now` defaults to `time.Now` and supplies wall time
only. The caller establishes clock trust; the coordinator validates shape and
ordering, not NTP state or the correctness of the host clock.

The renderer must implement `RenderReserved`: compose the complete logical
viewport, clear only `refreshstamp.Bounds(size)` before monochrome quantization,
and preserve source-free overlap warnings. `engine.Renderer` implements this
with its existing reserved-rectangle path. A renderer without this capability,
a missing timezone, or a display too small for the fixed font rejects at
construction. Legacy `NewScreen` remains unstamped and compatible.

One `ScreenPump` owns rendering, transport, and resolution. `ScreenDelivery`
borrows immutable output until resolution. A stamped delivery requires exact
`ResolveCycle(Cycle, confirmed)`; revision-only `Resolve` rejects because multiple
physical cycles can legitimately carry the same HTML revision. Cycle IDs never
wrap. They are local coordinator identities, not replacements for authenticated
transport transaction IDs.

## What changes and what does not

The coordinator retains canonical, **unstamped** confirmed pixels. New content
is compared with these pixels after the protected corner has been cleared.
Changing a timestamp therefore cannot defeat a no-op. The delivery gets a
separate owned copy on which `refreshstamp.Paint` draws the label and border.
It refuses occupied reserved pixels instead of hiding content silently.

`Current` is the accepted author revision and HTTP ETag. `Confirmed` is the
newest pixel-equivalent confirmed scene; `Delivered` is the revision in the last
actually acknowledged full delivery. A pixel no-op can advance `Confirmed`, but
not `Delivered`, the cycle ID, or the full-refresh timestamp. Warnings from the
new scene remain visible even for a no-op.

Maintenance uses a new physical cycle ID without creating an author revision.
When due, it renders the latest accepted pending scene even if optional author
debounce/max-wait has not expired. Its explicit lease is serialized with normal
queue leases; the remaining queued entry later becomes a pixel no-op. A newer
submission during rendering makes that result superseded before handoff. Once
`Send` starts, newer authoring cannot cancel an ambiguous physical transaction.

If that exact latest scene has already produced a source-free rejection
diagnostic, maintenance may refresh the older known-good baseline. Its delivery
still carries the older revision; the rejection remains visible, and the bad
revision is never marked confirmed. The queued rejected entry is consumed
without rendering it again. New authoring clears this remembered rejection.
An unclassified bare renderer rejection fails closed, without inventing a
diagnostic or entering a maintenance retry loop.

## Time and completion

The label is the candidate full-cycle **start** instant formatted in the
enrollment timezone. It is not the rendering timestamp, a heartbeat, or a
measurement of when pigment visibly settled. Only matching terminal protocol
success confirms that candidate. `FullRefresh` exposes historical cycle/start/
completion evidence; `RefreshTrusted` distinguishes usable evidence from
history retained after ambiguity or reset.

Maintenance becomes due 600 seconds (or the configured interval) after the last
successful completion, measured with elapsed monotonic time. The same pump
timer combines this deadline with author grouping and the device cooldown.
Maintenance bypasses optional grouping, **never device availability or its
minimum refresh interval**. Startup remains conservatively cooled down. Each
success starts a new interval from completion: a long offline gap schedules one
current full cycle, not a catch-up burst. These clock roles follow Go's
[monotonic-clock contract](https://pkg.go.dev/time#hdr-Monotonic_Clocks).

Failure/unknown completion never advances the confirmed timestamp. It invalidates
pixel proof; no generic send error grants permission to resend. The recovering
transport path retains an uncertain exact cycle until authenticated reconciliation
decides it. A reconciled success confirms that original candidate; its recorded
completion is when the manager learned the result, not a device wall timestamp.
An unconfirmed/reboot result invalidates proof and permits one fresh latest full
resynchronization after readiness/cooldown, without changing the author revision.

If wall time regresses or becomes invalid **after a real successful Send**, the
physical success still happened according to the terminal ACK. Resolution returns
a clock error, invalidates reusable proof, and does not advance the historical
timestamp. It does not claim that the physical cycle failed to occur. Negative
or regressing elapsed observations also reject, including observations older
than a concurrent author submission.

## Evidence and limits

`manager/screen_cycle*_test.go`, `screen_maintenance_test.go`, and
`screen_stamp_render_test.go` cover exact cycle identity, failed/late ACKs,
timezone pixels, independent wall/elapsed clocks, the before/at deadline,
continuous edits, forced-render supersession, explicit rejection fallback,
unavailable/long-offline no-burst behavior, counter/configuration bounds, copied
status history, and real engine protected-corner pixels/warnings. Pump recovery
and cooldown behavior is exercised separately in `screen_recovery*_test.go`.

This is software evidence. It does not replace the final manual USB/WPA3/panel
acceptance or prove a worst-case render, network, or physical refresh duration.
