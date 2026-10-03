# Stream final pixels into controller RAM

Date: 2026-09-05. Status: user-selected research direction, not implemented or
physically accepted. Target: verified 7.5-inch V2 panel, Driver HAT Rev2.3,
Pico 2 W. Do not transfer controller assumptions to another panel.

## Decision and evidence

**Decision:** investigate server → USB/authenticated Wi-Fi → bounded Pico chunk
buffer → SPI → display-controller RAM. Keep canonical scene and confirmed pixel
state on the server. Do not automatically increase the current receive buffer
to hold a full raw frame; the proposed additional approximately 16 KiB was not
approved. Preserve the current working firmware while proving streaming.

**Fact:** Waveshare's `epd7in5_V2.py` writes image data with commands 0x10 and
0x13 before issuing the separate display-refresh command 0x12. Its full-image
path supplies an inverted image to one plane and the image to the other.
Source inspected on 2026-09-05:
https://github.com/waveshareteam/e-Paper/blob/master/RaspberryPi_JetsonNano/python/lib/waveshare_epd/epd7in5_V2.py
This moving reference is orientation evidence, not an approved replacement for
our pinned driver. Pin and inspect the exact source/datasheet before changing
SPI sequencing, plane semantics or power behavior.

**Cross-check:** local official Pico source at commit
`c9bcd84db5adf5f085353649a8a5c31492bc5fb8`, `EPD_7in5_V2.c:313`, writes one
complete plane, inverts the source, writes the second complete plane, then
refreshes. This confirms sequential full-plane ordering for the candidate,
not arbitrary mid-plane switching or delayed network-fed writes. Project
logical polarity is adapted separately by the existing Go driver.
https://github.com/waveshareteam/Pico_ePaper_Code/blob/c9bcd84db5adf5f085353649a8a5c31492bc5fb8/c/lib/e-Paper/EPD_7in5_V2.c

**Inference:** this separation permits investigating chunked writes without
retaining a complete frame on Pico. A complete 800×480 1bpp plane is 48,000 bytes.
Removing actual frame/staging allocations can reduce SRAM use; an `io.Reader`
interface alone does not establish zero-copy, bounded allocation or throughput.

**Unknown:** peak RAM reduction, latency, energy per update, controller retention,
and safe interrupted-write recovery on our exact panel. Streaming may keep the
controller powered while waiting for the host; retransmitting planes or restoring
a partial-refresh base may increase traffic and energy. No performance/energy
improvement is accepted until measured against the working buffered baseline.

## Atomicity and ownership

This candidate changes the proposed atomicity boundary: controller staging RAM
may change before the entire transaction is validated, but **no refresh** may
start before successful complete validation. It does not preserve staging RAM
on failure. Existing validate-before-framebuffer-mutation requirements remain
in force for the current implementation; this explicit candidate must pass its
gates before becoming the production contract.

Authenticate each network chunk before forwarding it to SPI; also validate final
transaction completeness and integrity before commit. Bind chunks to session,
transaction, profile, plane, offset, length and target. Reuse the established
authenticated transport, never substitute a checksum for authentication. USB
remains physical trusted access with corruption/framing checks.

One writer owns the controller across all planes and chunks. One HTML revision
produces one immutable final target; never mix independently rendered patches.
Pico needs bounded transaction metadata, not a DOM. Server ownership of pixels
does not remove the need to identify sessions, retries and confirmed revisions.

## Failure and limit contract to prove

| Case | Required behavior |
| --- | --- |
| Truncation, invalid chunk, digest mismatch, cancellation | No refresh; invalidate staged transaction. Restore a complete known target before another commit. |
| Missing, duplicated, reordered or conflicting chunks | Require exact offsets, sequence and plane identity; reject ambiguity, never silently append or replay SPI writes. |
| Lost completion reply | Do not blindly refresh again. Reconcile matching transaction status; unknown state requires explicit resync. |
| Device reset, reconnect, controller sleep/power loss | Invalidate unproven controller RAM/base assumptions even when the visible image remains. E-paper image retention is not RAM retention. |
| USB preempts Wi-Fi | Abort incomplete network staging; never interrupt a physical refresh. New owner starts from known state. |
| SPI/BUSY/refresh failure | Report exact observable stage; no confirmed revision/timestamp advance. SPI write success cannot prove controller reception or visible output. |
| Slow sender or stalled host | Bound idle and whole-transaction durations; stop safely and release power using verified panel lifecycle. No indefinite powered wait. |
| Oversize or malformed input | Validate dimensions, plane count, byte totals, chunk sizes, offsets, padding, operation counts and decoded work with checked arithmetic. No allocation from unchecked lengths. |
| Retry storm | Bound retries and require resync/backoff; maintain panel refresh safety limits. |
| Partial update after RAM loss | Restore the required base/planes or use validated full refresh; never assume retained old-plane contents. Partial remains a separate acceptance gate. |

Freeze numerical transport limits only after resource probes; distinguish data
arrival deadlines from BUSY safety timeouts and refresh cadence. The server's
optional debounce/max_wait and 600-second maintenance policy remain separate.
Pico still waits for the server rather than refreshing autonomously.

## Design consequences and acceptance

- Two controller planes may require two server passes. Verify whether the exact
  controller allows the intended chunk ordering; do not assume plane switching
  resumes an address automatically. Count all retransmitted bytes.
- Without a local framebuffer, generic fill/mask/copy commands and base-frame
  reads may not be implementable efficiently. Negotiate only proven operations;
  server-side final opaque raster regions are the safe candidate fallback.
  Do not silently claim the draft mandatory mask/cache operations are supported.
- Keep the display-specific streaming adapter behind a capability boundary.
  Other panels may need the existing buffered path. No register numbers in the
  manager protocol, renderer or generic USB codec.
- Timestamp pixels belong to the immutable server target. Advance confirmed
  last-full state only after successful full-cycle completion; failed staging
  never advances it. Physical output remains a separate acceptance observation.
- Tests: truncate at every boundary; corrupt/reorder/replay chunks; lose ACK;
  reset between planes and before/after commit; inject SPI/BUSY failure; switch
  transport owner; reject resource excess; compare streamed planes against the
  buffered oracle at multiple dimensions. Assert zero refresh on invalid input.
- Measure TinyGo static/peak RAM, allocations, USB/Wi-Fi and SPI bytes, transfer
  and powered-wait time, total update energy, and successful visible output.
  Only then decide whether to replace the buffered path and revise its contract.

## Implemented isolated receiver slice

2026-09-05: `streamrx` implements a typed, single-owner full-image receiver;
`cmd/stream-contract-check` consumes two 800×480 passes with a 100-byte chunk
and a checking sink, without GPIO, USB or a full-frame allocation. This is not
the public binary protocol, SPI streaming adapter or deployable display firmware.
The working firmware, wiring, panel timing and receive buffer are unchanged.

State sequence: Idle → Receiving → Ready → Complete. Begin prepares a sink;
each pass verifies SHA-256 against the same immutable intent before proceeding.
Only explicit Commit from Ready may call the sink's refresh boundary. Input or
sink failures during staging call Abort and enter Failed. Close aborts incomplete
work and permanently closes this receiver. Completion is a sink result, never
independent proof of visible pixels or durable state across reboot.

Config fixes dimensions, one or two passes, fresh nonzero epoch, chunk maximum
(1–4096 bytes) and positive idle/total budgets. Those limits are prototype API
guards, not a frozen transport negotiation. Config must come from the actual
display capability, not unchecked peer claims. Row padding is checked before
SPI-bound writes. Logical pixel polarity is 1=black; the sink owns plane inversion.
The two-pass candidate sends 96,000 logical pixel bytes for our panel, excluding
headers/authentication, versus 48,000 with Pico-side replay from a framebuffer.
This is a bandwidth cost, not an asserted performance improvement.

Intent IDs increase within an epoch. Same completed intent replays status without
another Begin/Commit; conflicting or older IDs reject. Failed IDs cannot restart
implicitly: use a new ID and full upload. Only the latest intent is retained;
older outcomes and reboot/reconnect ambiguity require external reconciliation.
Epoch freshness, authentication, serialization, retry/cadence policy and wire
framing remain caller obligations, not implemented security guarantees here.
No caller chunk slice is retained; SHA-256 still keeps bounded internal block
state. Independent scoped review reported no required findings; it did not run
tests because its Docker socket access was denied. Test evidence above comes
from the primary agent's actual Dev Container runs.
The event-loop owner must call Tick while idle and Close on disconnect. Sink
methods must have their own bounded I/O/BUSY lifecycle; input deadlines cannot
interrupt a blocked synchronous sink call or an active physical refresh.

**Measurements:** focused race tests pass, receiver statement coverage 96.3%.
Host allocation test reports zero per successful transfer after construction.
The 100-byte-chunk checking probe builds for `pico2-w` with TinyGo: flash 69,972
bytes, static RAM 7,976 bytes. This includes its runtime/probe and is neither
peak heap/stack nor a before/after saving for complete firmware. The probe has
no radio, USB session, panel driver or hardware acceptance. No UF2 was flashed.
Repository quality task passes in 16 seconds; changed-line coverage 93.4%, total
88.5%; existing USB/Wi-Fi builds remain green. Untouched legacy length debt is
still reported, not suppressed.

Guards cover corruption in the second pass, all incomplete prefixes of a small
two-pass fixture, duplicates, wrong epoch/offset/pass, empty/oversized chunks,
padding, early commit, total/idle deadlines, clock regression, sink failures,
abort errors and lost completion replies. Full-size probe checks ordered bytes.
Remaining: real wire corruption/auth tests, controller plane replay oracle,
power/BUSY failures on actual adapter, reconnect owner integration, target peak
RAM/energy and visible acceptance. Next slice must bind the sink to the verified
controller sequence and framed USB owner, not advertise this core as ready USB.

## Follow-up: isolated USB/SPI binding

2026-09-05: the next slice now exists as `cmd/stream-device`, `panel.Stream`,
`streamwire`, `streamapp` and host `cmd/epaperstream`. See
[EPS1 contract, evidence and remaining gates](eps1-usb-streaming.md).
Quality task PASS (22 seconds); changed coverage 91.0%, total 89.3%.
Current target build: flash 96,152 bytes, static RAM 7,964 bytes. Neither peak
RAM nor energy savings is established. SPI wire equivalence and injected
failures are tested; visible target acceptance is still pending. No flashing.
The existing buffered firmware remains the recovery path.
