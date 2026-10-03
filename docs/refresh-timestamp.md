# Confirmed full-refresh timestamp

Historical integration checkpoint, superseded 2026-09-07: `refreshstamp` remains
shared, but the Blitz preview was removed. Current synthetic preview commands
and reservation semantics are in [native-preview.md](native-preview.md).
The original measurements and physical-acceptance gaps below are preserved.

Component and local preview verified, 2026-09-06. Manager-side component, not a new
USB protocol or permission to refresh the panel. Current EPS1 success does not
provide a physical-start timestamp, so it must not be silently substituted with
HTML receipt/render time. Follow `render-design-review-2026-09-05.md`.

## Contract for this slice

- `refreshstamp.Tracker` tracks one full cycle with a strictly increasing,
  nonzero cycle ID. Caller serializes access and binds IDs to its device session.
- `Begin` takes an explicitly trusted start instant and a previously supplied
  timezone; it produces a candidate label, not confirmation. No internal clock,
  NTP, timer, network access, retry or maintenance scheduler.
- `Complete` accepts only the matching cycle and a valid completion instant;
  only then does the confirmed label advance. `Abort` preserves historical
  confirmation but invalidates permission to use it as a partial-update base.
- Partial updates use only `ForPartial` after an unambiguous confirmed full
  cycle. Reconnection/unknown device state requires explicit invalidation and
  a confirmed full resync. This module cannot verify transport authentication.
- `Paint` uses the existing bitmapfont metrics/format (`YYYY-MM-DD HH:MM`) and
  borrowed `display.Frame`. It rejects invalid/too-small frames and non-white
  reserved pixels before mutation; no silent content erasure or full-frame copy.
  It cannot detect white-on-white HTML occupancy: layout-level reserved-region
  diagnostics remain a separate renderer integration gate.
- The old confirmed timestamp remains historical after failure; it is not
  proof of what pixels survived an interrupted physical refresh. No speculative
  completion time is painted, and unavailable time never falls back to UTC/now.

## Verification and local preview

Race-enabled tests pass with 100% statement coverage for `refreshstamp`:
candidate versus confirmed state, stale/duplicate IDs, failed/unknown cycles,
invalid time, repeated local DST hour, partial preservation, occupied corners,
small/non-byte-aligned/padded frames and unchanged pixels outside the rectangle.
An independent `font.Drawer` oracle checks every output pixel. `Paint` measures
zero allocations; this is not a whole-pipeline memory or energy measurement.
Independent code review found no required fixes. The task gate passes in 17s:
zero lint issues, changed-worktree coverage 93.5%, total coverage 90.1%, USB and
Wi-Fi TinyGo builds pass. Six pre-existing file-length exceptions remain visible;
no new exception was added. The two approved Blitz image-test ignores are unchanged.

The optional preview arguments are an operator-supplied **synthetic** instant
and IANA timezone. They never confirm a device cycle. Existing three-argument
preview usage remains unchanged. Stamp validation/occupancy errors occur before
opening the output file and preserve any existing artifact.

Inside the already-running Dev Container, from this experiment's directory:

```sh
go test -race -cover ./refreshstamp ./cmd/blitz-preview
go run ./cmd/blitz-preview ./blitz-probe/target/debug/epaper-blitz-probe ./testdata/blitz-image.html ./blitz-probe/target/preview-timestamp.png 2026-09-06T12:34:00Z Europe/Kyiv
```

Visual inspection of that real Go → BZR2 → Blitz → stamp → PNG run confirms
two images, readable Ukrainian text and `2026-09-06 15:34` in the bottom-right
120×21 rectangle. The fixture leaves this area empty; collision handling is
covered independently by tests. No USB transmission or panel refresh occurred.

## Remaining integration

Final integration still needs trusted start/completion metadata, provisioned
timezone binding, reserved-layout diagnostics and physical acceptance. The
600-second scheduler and existing 180-second EPS1 guard are unchanged.

## Sources

- Project timestamp semantics: `render-design-review-2026-09-05.md#timestamp-and-full-refresh`.
- Existing font/geometry: `../bitmapfont/metrics.go`, `../display/surface.go`.
- [Go time/location and clock semantics](https://pkg.go.dev/time): the caller
  supplies trusted instants; time formatting does not establish clock trust.
