# EPS2 USB host

2026-09-07. Candidate native Go tooling for `cmd/screen-device`. Software tests
do not establish USB reliability or visible panel output. No hardware acceptance
or flashing was performed for this slice. Keep the accepted EPS1 artifacts and
`cmd/epaperstream` unchanged; this tool has no EPS1 fallback.

## Commands

Build inside the already-running project Dev Container:

```sh
cd /workspaces/pico-sandbox/experiments/03-remote-epaper
CGO_ENABLED=0 go build -o build/epaperscreen ./cmd/epaperscreen
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o build/epaperscreen-macos-arm64 ./cmd/epaperscreen
```

On the computer owning the serial port, after explicit hardware acceptance is
authorized, the native executable supports:

```sh
./epaperscreen --status /dev/cu.usbmodemEXAMPLE
./epaperscreen -width 800 -height 480 scene.html /dev/cu.usbmodemEXAMPLE
```

Use the actual observed port, not the example name. The first command prints
numeric profile/version, dimensions, chunk/pass/cadence and network health.
It explicitly reports `transfer_evidence=unavailable`: Hello/Health replies
currently carry a zero transaction result, not live state/pass/offset or panel
failure evidence. Never interpret those zeros as Idle, no panel error, or no
image. No boot identifier, lease claim, content or credentials are printed.
Retain the original client for exact transaction Query/reconciliation; a new
status process cannot reconstruct a lost ACK or establish visible pixels.
It sends only Hello and Health, never Acquire, Bind, upload or refresh.
Opening USB/DTR still applies normal USB priority and can abort incomplete
Wi-Fi staging; status is not a promise of globally zero physical I/O.

The second command waits for an EPS2 connection and its full-refresh cooldown
before reading the latest regular HTML file and rendering it. HTML is limited
to 32 KiB and the pure-Go engine profile; assets follow the same engine policy.
Expected dimensions must match real advertised adapter geometry. Validated
profile IDs/versions are generic; host code does not implement panel registers.
An unenrolled device with zero DeviceID remains usable over trusted local USB.

Direct delivery uses the same protected bottom-right timestamp as the manager.
`-timezone Europe/Kiev` is the default; an explicit IANA name such as `UTC`
changes it. Zone data is embedded, so no system timezone package is needed.
The host clock is sampled **after readiness/cooldown**, at the beginning of the
full cycle. The label is not firmware installation time or proof of visible
pixels. Invalid zones fail before opening USB. A viewport too small for the
shared legible timestamp rejects rather than silently shrinking the text.
Source-free `warning=... element=...` lines report clipping/reserved overlap;
failure to write these diagnostics prevents submission. Lost completion still
uses the single-attempt reconciliation rule below, never a fresh timestamp retry.

There is one attempted Send. A lost reply keeps the same parent client alive
for Query/Abort reconciliation. Exact retained completion evidence can confirm
without additional SPI; an unconfirmed result exits with an error and does not
resend the old frame. A new process does not recover the previous process's
private lease or transaction; cold startup acquires fresh ownership and waits
the conservative full cadence. The CLI has a five-minute overall budget and a
15-second renderer budget. No boot image or autonomous device refresh is added.
An unconfirmed failure preserves the original exchange error, including
request operation/pass/offset, even when reconciliation also times out. These
coordinates identify the attempted request, **not** received bytes or an ACK.
No epoch, transaction ID, claim, digest or pixels enter these diagnostics.

## Manager interface

```go
sender, err := screenusbhost.New(nativeExecutable, serialPort, expectedSize)
// sender implements screendelivery.RecoveringSender and Close.
```

One ScreenPump owns WaitReady, Send and ResetSession. WaitReady negotiates,
checks exact retained capabilities/identity, and reconciles pending identity
before fresh rendering. Readiness.NotBefore is retained, not renewed by idle
polls. Boot/generation/profile changes return ErrTransportResync; the owner must
invalidate its pixel baseline before ResetSession and a fresh full cycle.
Send borrows the immutable manager frame only until return; the retained client
stores its digest/ID, not frame bytes. Close is required.

A fixed lifetime-scoped 15-second watcher only coalesces Changed notifications.
The ScreenPump performs the read-only Hello probe; the watcher never uses the
client or mutates Screen. This discovers an idle unplug/reboot even with no
author update. An unchanged probe does not acquire/bind or refresh. Cost when
idle and connected: at most one scheduled Hello/reply pair per 15 seconds
(32 request + 120 reply bytes); author-driven readiness checks can also probe.
An absent port retries connection at one-second intervals, without uploads.

## Bounded serial process

The paired `epaperscreen` worker and parent preserve a bounded diagnostic on
pipe EOF without forwarding stderr or raw packets. Worker exit20 means port
open,21 setup,22 serial write,23 serial read,24 reply validation. Other nonzero
exits produce an unknown-worker diagnostic; old exit1 workers remain usable.
These are local process categories, never controller completion evidence.

The parent waits at most one second after EOF for exit-status publication.
Successful reads do not incur this wait. EOF remains in the error chain, so
existing retry/reconciliation rules remain unchanged; no frame replay or reset
is introduced. A manager readiness retry may still recover without surfacing
the transient category in its HTTP status. Use the bounded `--status` command
for a single diagnostic attempt; it does not acquire a frame lease or refresh.
This improves diagnosis but does not establish the cause of post-flasher EOF.

References: [Cmd.Wait](https://pkg.go.dev/os/exec#Cmd.Wait),
[ProcessState.ExitCode](https://pkg.go.dev/os#ProcessState.ExitCode).

The parent owns the retained client and one long-lived child invocation:
`epaperscreen --serial-proxy SERIAL_PORT`. This internal mode must be supervised;
do not use it as a standalone serial sender. It opens one CDC port, asserts DTR,
and forwards only complete bounded EPS2 requests/replies. Each request and reply
passes Size/Decode/ParseReply; payload magic is ordinary payload data. The child
uses one 1056-byte scratch buffer, no renderer or framebuffer, and stdout is
binary only. Private lease claims are never command-line arguments or logs.

USB additionally caps each decoded data portion at **480 bytes**: the complete
32-byte-header record fits the 512-byte TinyGo 0.42.0 CDC RX ring. Each request
is acknowledged before another is sent, so correctness does not depend on the
owner draining a burst mid-arrival. PackBits includes its prefix and is selected
only when smaller than raw, so it cannot exceed this bound. Fixed control
requests are at most 64 bytes. Device-advertised MaxChunk remains unchanged;
effective USB size is `min(device MaxChunk, 480)`, including after reconnect or
ResetSession. Generic authenticated-network clients keep their negotiated limit.
There is no driver patch, extra Pico buffer, inter-packet sleep or relaxed
refresh floor. More requests/ACKs trade bandwidth for bounded safe admission;
energy/throughput improvement has not been measured. Recheck this assumption on
any TinyGo RX implementation/version change. Primary source inspected locally:
[CDC receive callback](https://github.com/tinygo-org/tinygo/blob/v0.42.0/src/machine/usb/cdc/usbcdc.go),
[512-byte ring](https://github.com/tinygo-org/tinygo/blob/v0.42.0/src/machine/usb/cdc/ring.go).

Each parent operation has a 180-second deadline, also bounded by caller context.
Cancellation closes parent pipes, kills and reaps the direct child. No child
spawns descendants. A one-second reap failure poisons that Sender: it cannot
replace or reuse an unreaped process. This isolates potentially blocking macOS
serial writes; serial.Close alone is not treated as a write deadline. Child
stderr is discarded rather than promoted to trusted diagnostics. Typed EPS2
status remains the diagnostic/evidence boundary.

Sources: [Go CommandContext](https://pkg.go.dev/os/exec#CommandContext),
[Cmd.Wait and pipe ownership](https://pkg.go.dev/os/exec#Cmd.Wait),
[Go serial v1.8.0](https://pkg.go.dev/go.bug.st/serial@v1.8.0),
[EPS2 contract](eps2-wire.md),
[controller-RAM boundaries](controller-ram-streaming.md).

## Regression evidence

Tests cover lost Acquire/Data/Commit replies without replay, two-pass odd-width
frames, retained cooldown, explicit reboot resync, read-only Health and idle
probes, stopped watchers, actual subprocess blocked read/write cancellation and
reaping, every truncated request prefix, CRC failures, payload-contained magic,
mismatched replies, no-progress I/O and serial setup/teardown failures.
The real sender and proxy also cross a 512-byte burst-limited RX fake into the
real EPS2 session: incompressible, mixed packed/raw and compressed 800x480
frames must deliver two exact 48,000-byte planes and exactly one Commit.
Before the USB cap, raw/mixed cases sent 1032-byte bursts and dropped 520 bytes;
compressed-only happened to pass. This reproduces a host contract defect,
not physical proof that it caused the first dashboard failure.
An actual DataPacked request/reply regression runs Hello/Acquire/Bind and both
compressed logical passes through the same proxy into the real EPS2 session;
it checks decoded bytes, successful replies and exactly one Commit.
Exact visible pixels, cable reconnect behavior and operating-system serial
driver behavior remain manual target acceptance, not inferred from these tests.

Timestamp integration: the missing `-timezone` regression first failed, then
passed with complete reserved-region pixel comparison for UTC, Europe/Kiev and
America/New_York, including a local-date boundary. Tests also prove the clock
is sampled after readiness and invalid clocks/zones/small viewports/diagnostic
write failures do not send. CLI race coverage: 93.2%; strict lint: zero issues.
