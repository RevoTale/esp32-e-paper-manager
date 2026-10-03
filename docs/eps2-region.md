# EPS2 region transaction

Status: implementation contract, not an enabled device capability. The accepted
ESP32 receiver remains full-only until lifecycle, sender, recovery and physical
acceptance gates pass. Existing full records and digest meanings are unchanged.
An opt-in candidate build now composes the complete region path for acceptance;
it is not installed on the device or enabled by default.

## BeginRegion

Reserve operation14 for a148-byte payload. Header epoch and transaction ID must
be nonzero; pass and offset are zero. All integers are little-endian. The frame
codec validates shape; the region codec validates content before any sink call.

| Offset | Bytes | Meaning |
| --- | --- | --- |
| 0 | 32 | Transaction digest |
| 32 | 32 | Last confirmed transaction digest (baseline identity) |
| 64 | 32 | SHA-256 of packed old-region canonical pixels |
| 96 | 32 | SHA-256 of packed new-region canonical pixels |
| 128 | 8 | Left, top, right, bottom as four u16 half-open coordinates |
| 136 | 1 | Priority: normal0 or urgent1 |
| 137 | 3 | Reserved, must be zero |
| 140 | 4 | Normal partial interval in whole positive milliseconds |
| 144 | 4 | Urgent partial interval in whole positive milliseconds |

Transaction digest is SHA-256 of ASCII `EPS2-region-v1` followed by one zero byte
and payload bytes32..147. It binds baseline, geometry, both plane hashes and
cadence. Commit/Query carry this digest, not a claimed full-image hash. Old/new
region hashes must be verified separately while streaming. No framebuffer is
introduced into the receiver. The zero baseline is invalid; an unknown baseline
requires full resynchronization, never optimistic partial rendering.

Generic geometry requires positive width/height, byte-aligned X endpoints and
bounds inside the negotiated display. A physical adapter must additionally
validate its own minimum dimensions, area and supported mode. One transaction
has one rectangle; manager edits are composed into one final target before
choosing its damage bounds. Unchanged pixels inside expanded bounds come from
that same target, including the preserved last-full-refresh timestamp.

## Receiver integration invariants

- Begin compares baseline identity with the last confirmed visible transaction
  under the same boot/owner epoch before invalidating staging or touching SPI.
- Pass0 contains old pixels and pass1 new pixels, each exactly rectangle
  width/8 × height bytes. Both use canonical1=black, independently of SPI polarity.
- Sequential offsets, matching IDs, digest verification and one writer remain
  mandatory. A valid transaction ID cannot be reused with changed metadata.
- Do not treat a successful Begin or upload as a visible update. Promote the
  new transaction identity only after controller completion. Unknown completion,
  lost baseline, owner change or reboot requires reconciliation/full resync.
- Interrupted staging must not refresh. Never replay a physical Commit merely
  because a response was lost. Existing Query/completion fencing remains required.
- Sender supplies old-region pixels from its confirmed immutable snapshot.
  Without a device framebuffer, receiver cannot independently prove that a
  supplied old-region crop matches that snapshot; authentication, baseline
  matching and sender replay-equivalence tests are required, not a false claim
  of pixel readback. CRC/digests are not authentication; EPN2 protects Wi-Fi.

References: [partial hardware research](esp32-partial-research.md),
[EPS2 compression](eps2-compression.md). Board enablement remains gated by
manager integration and physical acceptance; native tests may install the adapter.

## Go sender contract

`FeatureRegion` is capability bit4 and requires two passes. The Go codec accepts
it; C discovery advertises it only when a region adapter is installed before
lease acquisition. Only the opt-in candidate board runtime installs it. `SendRegion`
requires explicit advertisement; it never silently falls back to a full update.
The180-byte BeginRegion record includes its32-byte EPS2 header, independently
of the maximum data-chunk size.

`Client.Baseline()` exposes the last confirmed transaction digest under the
bound lease, or zero while disconnected, unconfirmed or pending. The manager
must retain the immutable pixel snapshot associated with that identity. The
client retains neither old nor new pixel slices. Admission checks metadata,
coordinates, both slice lengths/hashes, baseline and ID exhaustion before
consuming an ID. Full and region sends share chunking and commit verification.

After admission, a failed exchange keeps the pending identity and invalidates
the exposed baseline. Reconnect and Query resolve a lost response; no automatic
Commit replay occurs. Only confirmed completion installs the next identity.
Progress offsets are checked against the pending plane's byte count, not the
full viewport. A failed/unconfirmed result requires a full resynchronization.

### Sender checkpoint — 2026-09-30

Fact: focused Go tests exercise distinct old/new planes, one-byte chunks,
lost Commit response followed by Query with exactly one commit, unsupported
capability, invalid metadata and geometry, wrong hashes/lengths, exhausted IDs,
invalid Begin progress, and failures in both data passes. These are client-side
tests with a reply peer; they do not prove native C interoperability or pixels.

Regression: the existing progress guard accepted offsets outside a small region
because it used the full-frame length. A focused test first reproduced that
acceptance; pending plane length now bounds status validation. Strict lint also
caught excess function complexity; validation and terminal-state handling were
split without changing the configured limits.

Remaining: actual Go-client/native-receiver region interoperability, C feature
advertisement behind the installed adapter, manager snapshot/damage integration
and physical acceptance. No device enablement follows from these unit tests.

Measurement: `make quality` exited0 in the matching Dev Container, recorded in
`build/quality-region-sender.log`: strict lint, audit, race tests,24 native tests,
existing Go/C interoperability, TinyGo portable checks, ESP-IDF and Linux builds.
Three coverage runs passed the stability check at94.7%; changed executable
coverage was100%, including59/59 statements in the new Go sender. `git diff
--check` passed. No flashing, device frames, commit or push occurred.

## Native Go/C delivery checkpoint — 2026-09-30

Fact: the real Go client now interoperates with the C receiver and candidate
panel driver in the native executable. The explicit test-only `--region` option
installs the adapter; default invocation retains full-only behavior. Discovery
sets bit4 only with the installed adapter. Existing full-only clients and board
configuration retain their previous feature mask.

The test sends a48000-byte canonical full frame, followed by a16×2 region at
half-open coordinates(240,254)..(256,256). The native SPI seam independently
checks inclusive endpoint00ff, exact window bytes, full/partial plane polarity,
48004bytes per plane and exactly two refresh commands. GPIO, time, flash and SPI
are test seams; the client, EPS2 codecs, receiver hashing/lifecycle and panel
commands are actual implementation code.

A second test consumes and validates the genuine region Commit response, then
simulates losing it. The same client reconnects through Hello/Bind and confirms
the pending transaction using Query. Native plane/refresh counts remain
unchanged, proving this path does not resend pixels or refresh again. This is
lost-response recovery under a stable boot, not a physical power-loss test.

The new test first failed on missing capability advertisement. Focused Go/C
tests with race detection and `make interop c-size` subsequently passed. The
earlier sender checkpoint's missing-native-interoperability item is resolved
for these two scenarios; manager integration and physical qualification remain.

Measurement: full `make quality` exited0 after these changes
(`build/quality-region-interop.log`): all24 native tests, all Go/C interop,
strict lint, audit, race tests, stable three-run94.7% total coverage,100% changed
coverage, portable TinyGo checks, ESP-IDF and Linux manager builds.
`git diff --check` passed. No physical board, keys or running service changed.

Next integration boundary: `manager/screen_api.go` still rejects explicit
partial mode; `manager/screen_recovery.go` sends complete prepared frames through
`PolicySender`. Both `screenhub` and `screenusbhost` currently call the full-frame
client API. The next slice must carry confirmed snapshot/damage and refresh mode
through those boundaries, preserve timestamp pixels on partial updates, maintain
separate full/partial cadence, and invalidate the snapshot after unconfirmed
recovery. Do not remove the API rejection before that path is executable.

## Manager damage preparation

`screendelivery.PlanRegion` reuses the earlier `renderdiff` byte-row comparison
approach, specialized for EPS2's single rectangle. It takes complete immutable
base/target snapshots after composition and timestamp painting. It returns
owned old/new packed crops; it neither retains nor mutates the source frames.
Separated changes and erased pixels are included in the common bounds, not
sent as independently rendered or overlapping patches.

`RegionRules` supplies adapter minimum width/height and a per-plane byte budget.
Horizontal endpoints stay byte-aligned. Small damage expands within the viewport
using pixels from the same pair of snapshots; expansion at the right/bottom
edge shifts inward. A budget overflow returns `ErrRegionBudget` before allocating
crops, letting Auto select full refresh while explicit Partial remains explicit.
Identical logical pixels return empty bounds with no crop allocation. Stride
padding does not create damage. This EPS2 path rejects non-byte-width viewports
rather than silently cropping their last pixels; other adapters can use full.

Fact: this planner is a preparation module, not yet the live pump selection path.
The current Screen baseline excludes the reserved timestamp, so the pump must
paint the confirmed `Tracker.ForPartial` label into both prepared frames before
planning. A partial ACK must update pixel confirmation without calling the full
tracker's Begin/Complete or moving the600-second maintenance origin.

Regression tests reconstruct the final target pixel-by-pixel, validate old crops,
cover every edge, white erasure, separated damage, differing strides, immutable
inputs, no-op and pre-allocation budget rejection. A five-second fuzz run passed
427753 executions over varied byte-aligned viewports and pixel patterns. These
prove crop replay, not physical refresh support or measured energy savings.

Measurement: `make quality` exited0 (`build/quality-region-plan.log`), including
strict lint, audit, race tests,24 native tests, Go/C interop, TinyGo, ESP-IDF and
Linux builds. All three coverage runs reported94.7%; changed coverage100%,
including51/51 planner statements. `git diff --check` passed. No hardware action.

## Manager lifecycle and cadence checkpoint — 2026-09-30

Fact: opt-in `ScreenPartialOptions` now prepares old/new crops with the same
confirmed full-refresh timestamp painted into both snapshots. Confirmed partial
completion advances pixel confirmation without moving the full-refresh history
or maintenance origin. Consecutive partial count forces an Auto full update;
explicit Partial rejects instead of silently changing mode. Lost ACK retains
the exact pending lease: confirmed Query restores trust, otherwise the baseline
is invalidated. Clock rollback also invalidates confirmation.

Regression: explicit Full with unchanged content previously vanished as a
pixel no-op. It now requests a real full cycle. Tests first reproduced this,
lost-ACK trust not being restored, and clock rollback being accepted.
Full `make quality` passed in `build/quality-manager-partial.log`: three stable
94.6% coverage runs, changed coverage94.9%,24 native tests, interop, lint, audit,
race tests, TinyGo checks, ESP-IDF and Linux builds. This supersedes the earlier
statement that damage planning is not integrated into Screen preparation.

Decision: one shared transport `Cadence` holds separate full/partial and
normal/urgent deadlines, all measured from confirmed completion. Auto retains
the full deadline until an actual mode is selected. Unknown completion retains
the conservative full/reconnect floor and any longer existing partial guard.
Invalid configuration leaves the previous policy untouched. Reset revokes all
deadlines but preserves operator configuration; reconnect must establish a new
unknown-completion guard. Neither configuring a policy nor a zero readiness
field grants panel capability.

Fact: cadence tests cover the four lanes, unknown completion, longer operator
budgets, invalid configuration, and reset/reconnect. USB/Wi-Fi session resets
use the common reset operation. Transport region sends, pump selection of the
new deadlines, public API/CLI enablement and physical acceptance remain open.

Measurement: full `make quality` passed after the cadence changes, recorded in
`build/quality-partial-cadence.log`; `git diff --check` also passed. No hardware
refresh, flashing, service restart, commit or push occurred.

## USB and Wi-Fi adapter checkpoint — 2026-09-30

Fact: both transports now implement `RegionSender` using the existing single
client, lease, pending transaction and recovery path. Region and full delivery
share cancellation, transport-loss handling and completion cadence rather than
introducing another connection or retry loop. `ConfigurePartial` requires an
already configured full policy; otherwise a partial completion could lose the
full-refresh guard. Negotiation requires every configured capability, not only
the region bit. Missing support fails without a full fallback.

`RegionPlan.Wire` checks signed coordinates and packed lengths before narrowing
to u16, then binds the old/new hashes, current confirmed transaction identity,
priority and partial policy. Reconnection readiness is reconstructed after the
unknown-completion guard, including both partial deadlines; stale fast readiness
must not survive reconciliation. Current CLI/pump configuration still does not
enable this path automatically.

Measurement: a USB adapter/client test sends a full frame then distinct region
planes through a reply peer, verifies the shorter partial cadence and configured
wire policy, loses a Commit response, reconnects and resolves using Query with
exactly two total commits. The peer checks plane hashes and baseline identity.
This is a modeled peer, not physical USB or the C receiver. Wi-Fi tests verify
negotiation rejection, signed/length validation and the client capability guard
without another refresh. Successful encrypted Wi-Fi region delivery still needs
its own integration test; previous C interop covers the client/receiver boundary.

The initial full gate rejected excessive complexity, then insufficient coverage
(94.5%). Helpers now separate shape/capability validation, and delivery/recovery
tests exercise the actual adapter paths. No limits were lowered. Final
`make quality` exited0 (`build/quality-region-adapters.log`): three stable94.7%
coverage runs,95.1% changed coverage,24 native tests, Go/C interop, lint, audit,
race tests, TinyGo checks, ESP-IDF and Linux builds. No device action occurred.

Remaining integration: manager mode-aware scheduling, constructor configuration,
public API/CLI gates, successful Wi-Fi region/reconnect proof, and physical
qualification. Builds do not enable or accept the experimental panel adapter.

## Encrypted Hub/native receiver checkpoint — 2026-09-30

Fact: `TestEncryptedHubRegionToNativePanel` covers the real public Hub API,
EPN2 authentication/encrypted records, Go EPS2 client, native C receiver and
candidate panel implementation. An in-memory socket bridges decrypted records
to the native process; it does not simulate EPS2 replies. Device-side EPN2 is
the Go implementation in this test, not ESP-IDF radio/network code.

Both normal completion and a lost region Commit reply pass. The loss occurs
after reading the genuine native response. A new authenticated connection uses
the same receiver boot and durable-session model, and Hub reconciliation returns
`PendingConfirmed` without another upload or Commit. Native assertions prove
the exact window, plane polarity,48004 bytes per plane and exactly two refresh
commands. Post-reconciliation partial readiness retains the full uncertainty
guard. This closes the successful encrypted Hub region/reconnect test gap from
the preceding adapter checkpoint, not physical Wi-Fi or power-loss acceptance.

The native executable adds explicit test-only `--region-fast`: its advertised
legacy cooldown is1ms so the public host adapter can run without a180-second
wall-clock wait. Physical I/O/time remain native test seams; production firmware
configuration and the original `--region` fixture retain their prior settings.
The test has bounded operation/reconnect contexts. Its initial RED run lacked
the new fixture option and exposed an unbounded test wait; that own test process
was stopped, then the bounded RED reproduced the missing fixture before GREEN.

Measurement: focused tests passed with race detection; full `make quality`
exited0 (`build/quality-region-hub.log`), including all Go/C interoperability,
native tests, stable coverage gates, lint, audit, race checks, TinyGo, ESP-IDF
and Linux builds. `git diff --check` passed. No hardware, keys, service,
container lifecycle, commits or remote publication changed.

Next: wire mode-aware cadence into manager scheduling and configuration. The
planned mode is not always the actual mode: an Auto region may fall back to full
for area/count/maintenance reasons. The actual prepared delivery must meet its
own deadline; never transmit a fallback full frame at a partial-only deadline.
Prepared-but-unsent state must remain distinct from an ambiguous sent transaction
so readiness cannot accidentally reconcile or discard an unsent lease.

## Manager cadence accounting checkpoint — 2026-09-30

Fact: manager recovery now retains four independent monotonic cooldowns for
full/partial and normal/urgent. Confirmed completion starts each configured
budget; transport readiness can extend, but not erase, a longer local guard.
Missing partial readiness falls back to the full deadline. Unresolved Auto
still selects the full lane. Startup and uncertain recovery establish a full
safety floor for every lane without allocating timers or per-request workers.

Regression: a longer configured partial budget must restart after uncertainty,
not merely retain its old remainder. The new test reproduced30s remaining where
the operator required60s; recovery now takes the maximum of the prior remaining
guard, uncertainty floor and freshly restarted partial policy. Longer existing
transport floors also survive subsequent shorter postponements.

This is cadence accounting, not live partial acceleration: construction still
does not configure the pump partial policy, and actual-mode dispatch gating is
the next integration step. Keep public API/CLI enablement closed until an Auto
fallback full waits for its full deadline, prepared-unsent leases cannot be
mistaken for sent ambiguity, and superseding author work remains bounded.
Full stamp creation currently occurs during preparation; delayed dispatch must
also avoid labeling a later refresh with an early preparation timestamp.

Measurement: focused manager race tests and full `make quality` passed
(`build/quality-manager-cadence-final.log`, exit0): three identical coverage runs
at94.7%, changed coverage95.5%,24 native tests, Go/C interoperability, lint,
audit, race tests, portable TinyGo checks, ESP-IDF and Linux manager builds.
No device, running service, credentials, commit or remote branch changed.

## Mode-aware manager dispatch checkpoint — 2026-09-30

Fact: `ScreenPump.ConfigurePartial` now explicitly configures a
`RegionPolicySender` after the full policy and before Run. It requires Screen
geometry/count limits and does not silently enable a non-negotiating transport.
Failed configuration retains the prior pump policy. Public HTTP/CLI opt-in and
physical board adapter enablement remain separate, unfinished integration.

The pump may prepare eligible Auto work at the earliest applicable deadline,
but dispatch always checks the actual prepared mode. Area-budget fallback full
therefore waits for the full deadline even when rendering began at a partial
deadline. Count exhaustion and due maintenance select the full lane beforehand.
Only one prepared snapshot is retained; there is no polling worker or replay.

Prepared-unsent state is distinct from sent/ambiguous state. NoPending readiness
preserves an unsent lease and can extend its deadline; a claimed confirmation
for unsent work is rejected. New author work supersedes an unsent snapshot,
including work arriving during a blocking readiness call. A final pre-dispatch
check also replaces stale maintenance unless the latest scene was explicitly
rejected, in which case the existing known-good fallback remains permitted.

`Tracker.DiscardUnsent` preserves the previous confirmed label and consumes the
discarded ID permanently. Only proof of no device I/O permits this operation;
sent failures still invalidate trust. Cancellation before transmission preserves
the baseline. Reset invalidates it and requires full resync. A partial lease
that waits past maintenance is replanned as full. Delayed full stamps are rebuilt
from the unstamped candidate at dispatch without rerendering HTML; immediate
cycles retain their original IDs. Invalid clock/tracker state stops before I/O.

Regression evidence: RED reproduced full-only cadence for small changes,
fallback full sent at a partial deadline, stale author/maintenance delivery after
readiness, and invalid confirmation destroying an unsent lease. Deterministic
pump tests now observe full/partial/partial/full at30s/32s/34s/64s. Separate area
fallback tests prove preparation before the full deadline, actual full dispatch
at60s with the updated timestamp, and cancellation preserving confirmation.
Readiness, reset, maintenance, bad configuration, clock and cycle guards pass
with race detection. These times are test clocks and operator policy, not a
measurement or safety qualification of the physical panel.

Full quality passed on2026-10-01 (`build/quality-manager-dispatch-stable.log`):
three identical covered-block profiles at94.8%, changed coverage95.4%,24 native
tests, Go/C interoperability, TinyGo portable checks, ESP-IDF and Linux builds.
The earlier `final` run stopped on a vulnerability-database network EOF; the
`retry` run exposed a TCP cancellation test race. `Dial` did not prove that
`Serve` had accepted the socket, admitted it and registered its worker, causing
different Accept/worker coverage. A test listener now signals the second Accept
before cancellation. Twenty focused race runs passed, followed by the full gate.
No production TCP code, sleeps, skipped assertions or thresholds were changed.
No hardware frame, flashing, service restart, key change, commit or push occurred.

## Partial rejection checkpoint — 2026-10-01

The manager previously returned the planner's `ErrUnconfirmed` or
`ErrRegionBudget` through the pump when an author explicitly required partial.
That stopped the service even though no device I/O occurred. The new regression
first failed with `refreshstamp: full resynchronization required`.

`ErrPartialUnavailable` now classifies this bounded per-scene rejection while
preserving the original cause for Go callers. Status exposes a separate
`refresh_failure` revision/reason, not a fabricated HTML diagnostic. The pump
consumes the rejection, preserves its confirmed pixels and full-refresh history,
and waits for new work. Maintenance uses the older accepted scene; it must not
silently promote the rejected target as full. A new accepted submission clears
the diagnostic. Queue ownership failures remain fatal rather than being hidden.

Tests exercise continued delivery, maintenance revision, both rejection reasons,
source-free diagnostics, copied status ownership and queue-failure precedence.
Public API/CLI enablement and physical partial qualification remain separate.
`make quality` passed (`build/quality-partial-rejection.log`): identical covered
blocks in three uncached runs at94.8%, changed coverage95.5%,24 native tests,
race tests, strict lint, vulnerability audit, Go/C interoperability, TinyGo
portable compilation, ESP-IDF firmware and Linux amd64/arm64 manager builds.
`git diff --check` passed. No flashing or hardware acceptance occurred.

## Codec verification checkpoint — 2026-09-30

Fact: Go and native C decode the same region payload, including the domain-bound
digest, coordinates, baseline, old/new hashes and operator cadence. Interop
compares reconstructed fields, not only an echoed input. Tests reject every
single-byte mutation, truncated payloads, reserved fields and invalid geometry.
The C decoder leaves its output unchanged on rejection.

The full-only receiver explicitly returns configuration error12 for operation14.
Regression checks cover idle, active upload and staged states: rejection does
not consume an ID, abort staging or invoke panel I/O. The initial test failed
because dispatch returned unknown-record error1. A separate old health test
still treated14 as unassigned; its unknown-operation assertion now uses15.
Neither correction enables physical partial refresh or advertises it.

Measurement: `make quality` exited0 in the matching Dev Container. All22 native
tests and Go/C interoperability passed; three independent coverage profiles
were identical, total coverage94.6%, changed executable coverage100%. TinyGo
portable-package compilation, ESP-IDF application and both Linux manager builds
passed. Local evidence: `build/quality-region.log`. No CI, flashing or physical
refresh was performed for these changes.

Next: integrate negotiated region reception, independent pass hashing and
baseline lifecycle with the panel window adapter, then sender/manager replay
equivalence and fault recovery. The physical acceptance gates remain open.

## Receiver lifecycle checkpoint — 2026-09-30

Fact: `screen_region.c` now admits region transactions through an optional
`ep_region_sink`, configured before the first lease and only for two-pass
profiles. Its validation callback is pure; its begin callback shares the full
sink's context and write/commit/abort lifecycle. No physical adapter installs
these callbacks yet, and discovery still does not advertise regions.

Admission checks geometry, panel validation, the latest confirmed transaction
identity and selected cadence before consuming an ID or touching the sink.
Accepted staging invalidates the confirmed base. Each pass uses its own digest
and the region byte count, not the full-frame byte count. Only a successful
commit establishes the next baseline. Corruption, timeout or disconnect requires
a full resynchronization; a failed driver operation retains the fatal fence.
Duplicate completed commits return the existing result without another refresh.
The receiver adds bounded metadata, not a framebuffer.

Regression tests cover distinct old/new bytes, stale baseline, active admission,
invalid metadata, panel-specific geometry, urgent cadence, both corrupted planes,
disconnect, timeout, begin/write/commit/abort failures, full resynchronization,
duplicate commit and an area smaller than the display. The new timing test first
failed because operation14 was missing from synchronous-sink time accounting;
it now follows Begin/BeginRefresh without extending the budget through queries.

Measurement: `make quality` exited0 (`build/quality-region-lifecycle.log`):
23 native tests, Go/C interoperability, identical three-run coverage94.6%,
changed Go coverage100%, TinyGo compilation, ESP-IDF and Linux builds. A later
test-only addition checks region-versus-full byte limits; its native/C-size
verification is recorded separately in `build/region-extent-check.log`.
No device was flashed and no hardware refresh was requested.

Remaining: physical window/plane implementation, negotiated sender/manager
integration, replay-equivalence and physical fault/refresh qualification. The
generic receiver tests do not establish controller RAM behavior or pixel output.
