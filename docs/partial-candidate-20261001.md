# Partial candidate — 2026-10-01

Status: software integration candidate, not physical or production acceptance.
The accepted full-only checkpoint is [recorded separately](hardware-acceptance-20260930.md).

## Composition

Manager HTML/inline CSS → final opaque pixels → one coherent damage rectangle →
EPS2 old/new region planes → authenticated transport → receiver → guarded panel
adapter. No controller framebuffer or HTML parser was added to ESP32.

`-experimental-partial` is opt-in and requires `-screen -refresh-policy` plus
an800×480 viewport. The public API accepts explicit partial only after transport
configuration succeeds. The transport still negotiates region support with the
device: configuration is not a promise that a connected full-only receiver can
perform it. Existing old-firmware and invalid-baseline rejection gates remain.

Operator budgets:1s normal/urgent partial,5 consecutive updates,12000 bytes per
plane by default; each is configurable. These are not manufacturer safety
certification or measured device timings. Full cadence and10-minute maintenance
retain their existing configuration. See [API](screen-api.md).

The ESP32 candidate is built with `CONFIG_EP_EXPERIMENTAL_PARTIAL=y`. Its region
callbacks pass through the same RTC guard as full refresh. Validation must be
side-effect-free; the guard is set before begin, retained after failed begin or
cleanup, and cleared only after successful commit/abort. The wrapper forwards
the original panel context, not its own null context.

Memory basis: [ESP-IDF5.5.5 RTC memory](https://docs.espressif.com/projects/esp-idf/en/v5.5.5/esp32/api-guides/memory-types.html#rtc-slow-memory).
This preserves the existing guard design; host simulation does not establish
retention after every real reset/power fault. Panel waveform/geometry evidence
and remaining wake/SRAM uncertainty are in [hardware research](esp32-partial-research.md).

## Build and verification

Run in the existing Dev Container:

```sh
make quality
make firmware-partial
make latency
```

The candidate uses `firmware/esp32/build-partial`; ordinary `make firmware`
uses `build-release`. Neither command flashes. The separate sdkconfig/defaults
keep candidate opt-in out of the recovery configuration.

RED evidence: HTTP partial returned501 even after transport configuration;
CLI rejected the absent flag; native compilation identified the absent guarded
region interface. Focused Go race/lint and native guard tests passed after
integration, including invalid cadence, incompatible viewport, metadata bounds,
context forwarding and failed cleanup retention.

Measurement: full quality and the separate candidate build passed. The quality
target now includes both firmware variants to prevent the opt-in startup branch
escaping compilation. Log: `build/quality-partial-enablement.log` (prior target
composition) and `build/firmware-partial-candidate.log`. Three uncached coverage
runs matched at94.8%, changed coverage95.4%;24 native tests passed. The final
combined-target `make quality` also exited0:
`build/quality-partial-candidate-final.log`, including both ESP32 variants and
Linux amd64/arm64 manager builds. No assertions or thresholds were weakened.

`make latency` passed in this Dev Container: read119062µs/write68969µs against
120000µs, cancellation26671µs against150000µs. The read sample is near its
ceiling; this is one host sample, not ESP32 timing qualification.

Candidate application:774304 bytes, SHA-256
`065ed939158732798b2a67bf569ef2c7f5f869c668b0e389a819a1877da7dda3`.
`idf.py size` reports static DRAM36484, IRAM107867 and RTC SLOW64 bytes;
these are linked sections, not peak RAM, free heap or measured stack headroom.
Local report: `build/partial-size.log`. Verified sdkconfig: partial=y in
`build-partial`, unset in `build-release`.

Physical full→partial→full output, sleep/wake, ghosting, runtime
resources, transport faults and startup EOF remain unqualified. No flash,
hardware frame, service restart, credential mutation, commit or push occurred.

## Next authorized hardware boundary

Before flashing, obtain explicit permission for native host USB inspection,
application-only write and bounded test frames; confirm this exact ESP32/7.5 V2
is connected. Recheck serial ownership and preserve the existing known-good
artifact and private configuration. Never erase all flash or write credentials,
partition table or bootloader as an implicit part of this test.

After verified application write, inspect EPS2 capabilities and baseline state.
Do not repeat flashing merely because first serial status returns EOF; capture
bounded startup evidence instead. Begin with one known full image, then small
partial damage crossing an X endpoint of256, then another changed region after
the normal sleep/wake lifecycle. Verify unchanged pixels and unchanged full-time
corner. End with a full update and compare visually to its exact server bitmap.
Record transaction/cycle identities and measured timings separately from pixels.

If the user sees corruption, the wrong image or unexpected persistence, stop
partial testing. Host ACK is not visual proof. Recovery to the saved full-only
artifact also requires write authority; do not alter wiring as a software-debug
shortcut. Physical fault injection, resource/power measurements and public
distribution approval remain separate explicit boundaries.

Read-only independent review found no blocking issue in CLI/API configuration,
guard context/lifecycle or candidate composition. A stale full-only description
was corrected. This review did not execute hardware or expand the evidence of
the tests above.

## Authorized hardware attempt — 2026-10-01

The no-flash statement above describes the earlier software checkpoint. The
user subsequently authorized application-only flashing and full → partial →
full testing on the connected ESP32/7.5-inch panel.

The candidate identified above was written at `0x10000`; the native flasher
reported successful MD5 verification `42c058f31c1fedc5874aad5d247cb126` and reset.
The flasher updates the embedded image digest, so this written-image MD5 is not
the input-file SHA-256. Bootloader, partition table and credentials were not
selected for writing.

Two initial status attempts returned EOF. A temporary USB manager accepted
revision1 but remained current1/confirmed0/delivered0, with no in-flight frame
and untrusted refresh state, for more than four minutes. It was stopped to
prevent a delayed test frame. No partial frame was submitted.

Subsequent standalone USB inspection succeeded:800×480, two passes, chunk1000,
minimum-full180000ms, remaining0 and uptime400s. Panel trace reported state0,
cycle0, phase0, step0 and zero BUSY sample counts: no panel cycle was recorded
in this boot. This does not prove a wiring fault or establish the EOF cause.
The startup/transport diagnosis remains open; successful flashing alone is not
display acceptance.

After the successful standalone inspection, a second temporary manager run
delivered revision1/full cycle1. Reported UTC timestamps were
`2026-09-30T21:50:43.65488Z` → `2026-09-30T21:50:58.809721Z` (15.155s;
the local calendar date was October1). Status became
current1/confirmed1/delivered1/in-flight0, refresh-trusted=true. The image is
`ESP32 PARTIAL CHECK`, `FULL A`, with a black16×64 rectangle at(240,180).
The manager retains the180s unknown-state transport guard even when configured
with30s normal/urgent full cadence. The earlier unsuccessful run remains
unexplained; this later success is not evidence that the startup EOF is fixed.
The user confirmed the first full image visually.

Revision2 explicitly requested partial and changed only that rectangle to
white through a by-ID style edit. Status became current2/confirmed2/delivered2,
in-flight0 and refresh-trusted=true; full-refresh cycle1 and its timestamps
remained unchanged. The user confirmed that the rectangle disappeared and the
remaining image was unchanged without visible distortion. This qualifies one
small black-to-white partial on this board/panel, not all regions or endurance.
No exact partial duration was captured.

Revision3 explicitly requested full and restored the black rectangle. Status
became current3/confirmed3/delivered3, in-flight0 and refresh-trusted=true.
Full cycle3 ran `2026-09-30T21:55:56.424019Z` →
`2026-09-30T21:56:11.531324Z` (15.107s). The user confirmed the final full
visually: the black rectangle returned and the remaining image was correct.
This completes one USB full → partial → full hardware sequence, with protocol
confirmation for all three revisions and user visual confirmation at each step.
It does not close startup EOF, repeated-region/endurance, Wi-Fi partial,
power-fault recovery or production resource qualification.
The temporary manager was sent SIGTERM after terminal confirmation
to prevent unrequested background maintenance frames. No additional flash,
credential change, commit or push was performed during these frame tests.

## Startup triage after the accepted sequence

Measurement: a subsequent standalone Hello/Health succeeded with uptime1212s,
remaining0, network state6/last-failure3/failures11. This single sample does not
identify the network failure or establish general Wi-Fi health. No frame or
flash was issued during this check. The test manager and worker were no longer
running when checked before proposing a manual cold-start test.

Fact from source: `screenusbhost/process.go` discards child stderr and the
`cmd.Wait` error; a failed proxy can therefore surface only as pipe EOF.
`proxy.go` exits on serial setup, framing, read/write or reply-validation errors.
EOF alone cannot distinguish these causes. Application and bootloader logging
are disabled in the candidate sdkconfig; this does not rule out ROM/driver
startup effects. No change to framing, retry rules, timing or DTR was made on
this evidence.

Unknown: the initial post-flasher EOF is not currently reproducible with a
warm read-only query. A cold-power startup check is the next controlled
experiment, distinct from reproducing the original post-flasher reset. Preserve
the successful candidate and credentials; do not reflash as a diagnostic retry.

Cold-start measurement: after the user removed/reconnected power, the first
standalone USB Hello/Health succeeded, uptime21s and remaining158968ms. No EOF
was observed in this sample. It verifies this cold startup only, not the earlier
post-flasher reset sequence. Network state6/last-failure3 means backoff after
the TCP-dial stage in `platform/network.c`; it is not evidence of a wrong
Wi-Fi password. No manager was listening before this check.

The subsequent temporary manager uses Wi-Fi-only transport and the existing
September30 enrollment. The test submits `WIFI PARTIAL CHECK` with a black
rectangle at(240,180),16×64, as revision1/full. A device-link TCP connection was
observed; frame confirmation and visual acceptance must be recorded separately.
No credentials, endpoint configuration or firmware were rewritten.

Wi-Fi full protocol result: current1/confirmed1/delivered1, in-flight0,
refresh-trusted=true. Cycle1 ran `2026-09-30T22:07:58.183736Z` →
`2026-09-30T22:08:04.074253Z` (5.891s). Manager RSS at the status sample was
28144KiB, not a peak or receiver-RAM measurement. The user confirmed the full
image visually before the partial mutation.

Wi-Fi revision2 explicitly requested partial and changed only the probe's
background to white. Status became current2/confirmed2/delivered2, in-flight0,
refresh-trusted=true. Full cycle1 and its timestamps remained unchanged. The
user confirmed the rectangle disappeared with the rest of the image unchanged
and without visible distortion. This establishes one small black-to-white
partial over the encrypted Wi-Fi transport on the same candidate, in addition
to the accepted USB sequence; it is not endurance or general-region acceptance.
Exact partial duration was not captured. No USB image transmission was used.

After visual confirmation, the temporary Wi-Fi manager was sent SIGTERM to
prevent further background maintenance. Firmware, keys and enrollment remain
unchanged. Startup post-flasher EOF remains unresolved despite the successful
cold-start sample. Runtime receiver resource/power, repeated-region/ghosting
and physical fault-recovery qualification remain open.

## USB diagnostic preservation — 2026-10-01

Confirmed software defect: the parent discarded worker exit details, so an
unopenable serial port and a failed serial exchange both surfaced as pipe EOF.
The worker now emits fixed exit categories20–24 (open/setup/write/read/reply),
and the parent attaches the safe category after EOF. Raw stderr stays discarded.
Exit-status collection is bounded to1s; normal reads do not wait. Existing EOF
classification and replay/reconciliation policy remain unchanged. These codes
identify a failed stage, not a hardware root cause or completion result.

Tests cover error categories, cause preservation, successful/unknown exits,
bounded exit publication, subprocess EOF and private-stderr non-disclosure.
The initial focused test compilation failed because the diagnostic interface
was absent; the implementation then passed focused race tests and lint.

Native macOS before/after reproduction used the same nonexistent test port:
the retained old client returned only `EOF`; the new client returned `EOF` plus
`screen USB: serial-open failed`. A read-only query to the actual ESP32 then
succeeded with uptime1210s and remaining0. No image, reset, flash or credential
mutation was requested during this diagnostic check. This fixes lost software
diagnostics; it does not resolve the original post-flasher EOF cause.

Verification: `make quality` exited0, recorded in
`build/quality-usb-diagnostics.log`. Three uncached coverage profiles matched:
94.8% overall,95.8% changed-code coverage;24 native tests passed. Race/vet/lint,
workflow checks, vulnerability audit, interoperability, TinyGo portable builds,
both ESP32 variants and Linux amd64/arm64 builds passed. This run did not flash.
The native diagnostic client is `build/epaperscreen-diagnostic-darwin-arm64`.
Self-review checked EOF retry compatibility, synchronization through the closed
done channel, bounded exit collection, old-worker fallback and secret-safe
output. No dependency, EPS2 format, DTR policy or panel behavior changed.

## HTML list soak — started 2026-09-30 UTC / 2026-10-01 Kyiv

User approved a30-minute Wi-Fi test with one list row replaced every20s.
Five rows rotate; each replacement contains a current Kyiv date/time and an
incrementing number. Existing auto-refresh policy remains: at most5 consecutive
partial updates before a full refresh. This is an experimental test, not a
panel lifetime or production claim.

Temporary runner source/log: `build/items-soak/main.go` and
`build/items-soak/run.log` (ignored artifacts). Container gofmt/vet and the
native macOS cross-build passed. Manager PID60483, runner PID60489;
initial RSS samples24976KiB and11744KiB respectively, not peaks. First full
revision1 submitted at2026-09-30T22:33:23Z; confirmation was still pending
when this entry was written. The30-minute schedule starts only after that
confirmation, then sends89 updates; delayed execution never catches up in a
burst. Each update requires confirmed=delivered=current, no in-flight frame,
trusted refresh state and no reported failure before continuing.

Heartbeat `esp32-html-items-20s-soak` monitors the runner every2min; the native
runner owns the20s cadence. It must stop these temporary processes on completion
or failure, pause itself, and record actual results. No firmware, credentials,
USB state or production service was changed. Completion and visual acceptance
remain pending; START/END and CONFIRMED log lines are the timing authority.

### Stopped on confirmation deadline

Measurement: first full completed2026-09-30T22:36:58.223301Z (cycle1,
started22:36:50.460499Z). Runner START22:36:58Z, intended END23:06:58Z.
Revisions2–6 were submitted every20s and confirmed at22:37:20,22:37:39,
22:38:00,22:38:19 and22:38:40Z; each retained full cycle1. Revision7 was
submitted22:38:58Z but exceeded the runner's18s confirmation deadline.
Runner exited without replay; the30-minute test did not complete.

Read-only status after that exit showed current=confirmed=delivered=7,
in-flight0 and refresh-trusted=true. Full cycle7 started22:39:09.448635Z
and completed22:39:19.467296Z (10.019s physical-cycle interval), approximately
21s after submission. Observed full cycle identifiers were1 and7; this does
not mean seven full refreshes. The later confirmation disproves interpreting
the runner timeout as a permanently failed frame. Why the full cycle started
roughly11s after submission remains uninvestigated; do not assign a hardware
cause or silently lengthen deadlines and call the original test successful.

RSS snapshots: manager24976→37360→37392KiB; runner11744→13776KiB before
exit. These samples are not peaks or proof of a leak. Manager PID60483 was
verified, sent SIGTERM, and its exec session exited0. Heartbeat was paused;
no restart, extra frame, firmware or credential change followed. User observed
occasional black/white flashing during the run, not full visual acceptance
of every revision. Next investigation must separate scheduling delay from
refresh duration and test-runner confirmation budget.

### Diagnosis and second run

The user subsequently confirmed the final revision7 list visually. Source
trace resolves the scheduling delay: `screenhub/delivery.go` calls
`cadence.Complete` after every successful send, including partial. In
`screendelivery/cadence.go`, this sets the full deadline to latest completion
+ configured30s. Firmware `core/screen_timing.c` records every commit completion
in `last_refresh`; `core/screen_frame.c` enforces that interval for full begin.
The last partial finished approximately22:38:39Z; full became eligible around
22:39:09Z, matching the measured start. No firmware fault is demonstrated.

Added deterministic regression `TestFullAfterPartialUsesLatestCompletion`.
Test-runner budget/schedule tests failed before correction, then passed:
60s confirmation budget (30s configured cooldown +30s delivery allowance),
skip missed20s slots without burst catch-up, retain sequential row numbers.
Last minute is reserved for completion/observation, so total submissions vary.
This is a test budget, not a new physical-refresh guarantee. No device or
manager refresh policy changed. Focused race tests, runner vet/cross-build and
full root `go test ./...` passed; full `make quality` was not rerun this turn.

Second run first full submitted2026-09-30T23:01:05Z. Native manager PID62509,
runner PID62514; RSS baseline24656/11888KiB. Log `build/items-soak/run-v2.log`
preserves this run separately from the first failure. Heartbeat resumed with
new PIDs, actual-count semantics and hard deadline23:40Z. START/END and final
confirmation/visual results remain pending; do not infer success from launch.

### Second run interrupted: wall-clock continuity lost

Observed START23:04:17Z, intended END23:34:17Z on September30. Actual
confirmations continued through October1 00:20:14Z. Revisions1–32 all have
confirmed=delivered=current, in-flight0 and trusted=true in the log. Six
distinct completed full cycles were observed:1 at23:04:17.042156Z,
7 at23:06:34.218017Z,13 at23:08:34.646705Z,19 at23:29:25.567809Z,
25 at00:02:51.650690Z and31 at00:20:08.856789Z. Each recorded full-cycle
interval was approximately5.6–6.3s. These are protocol observations, not
visual verification of32 images.

Large unexplained execution gaps invalidate the continuous30-minute test:
revision19 submitted23:14:11Z but confirmed23:29:25Z; further approximately
16-minute gaps followed. The runner emitted neither COMPLETE nor TEST FAILED.
Its intended wall-clock end and the heartbeat hard deadline were exceeded.
The heartbeat itself was delayed: an invocation timestamped23:14:01Z obtained
later outputs through00:20Z. Native clock then reported00:20:16Z. Suspected
host sleep/suspension or clock discontinuity is not confirmed by power logs.
Do not attribute these gaps to panel refresh, or call this endurance acceptance.

RSS observations: manager24656→36880→36928→36272→36304KiB;
runner11888→13696→13984→14064→14240KiB. Not peak/RAM-leak evidence.
Exact executable/PID checks preceded SIGTERM of runner62514 and manager62509;
both were absent afterward, manager session exited0 and runner pipeline143.
Heartbeat paused; no automatic restart or new frame was issued by monitoring.
Next test needs an explicit host-awake arrangement and independently validated
wall-clock expiry across suspension. The current timer-based runner must not
be reused as proof of strict30-minute wall-clock bounding.

### User acceptance boundary

The user accepted the current working prototype as sufficient to proceed and
waived another endurance run on October1. This does not turn the interrupted
test into a pass. On October3 the release direction was clarified: one firmware
with full/partial capability and one corresponding manager version, not two
firmware products. Release packaging preserves those limits and does not flash
the board or remove the existing operator refresh safeguards.
