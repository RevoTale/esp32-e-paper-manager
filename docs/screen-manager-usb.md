# Server-rendered USB manager — experimental

Historical Blitz/EPS1 entry point, superseded 2026-09-07: `-render-worker` and
the Rust adapter below were removed. Current manager/USB tools use the native
engine; use [engine-authoring-api.md](engine-authoring-api.md),
[native-preview.md](native-preview.md) and [the removal map](blitz-removal-map.md).
The older protocol measurements below are preserved, not current acceptance.

Updated 2026-09-05. Opt-in runnable path, not the final public service or Wi-Fi
firmware. The existing legacy manager remains available unchanged.

```text
trusted author → loopback HTTPS → Screen / renderbatch → Blitz worker
               → one ScreenPump → epaperstream --raw → EPS1 USB → controller RAM
```

## Implemented scope

- One explicitly configured display, complete UTF-8 scenes up to 32 KiB.
  No device allocation from request paths or renderer per HTTP request.
- Existing bearer check and 30 submissions/minute process-local budget.
  Authentication precedes lookup/body reads. No CORS expansion.
- Conditional replacement, one active delivery and one newest pending scene.
  Optional debounce/max_wait preserve the agreed zero semantics.
- CPU Blitz rendering, bounded local BZM1 frames, EPS1 two-pass delivery.
- No idle polling: one wake channel and a timer only for pending work.
  No automatic retry or persistent replay after an ambiguous result.

## Run

Build in the existing Dev Container. The manager and USB worker run on the host
owning the serial port; Blitz may use the existing container wrapper. Starting
a macOS host server requires explicit user permission under workspace rules.
No service is installed, autostarted or publicly deployed by this increment.

```sh
epaper-manager \
  -tls-cert /private/config/localhost.crt \
  -tls-key /private/config/localhost.key \
  -token-file /private/config/api-token \
  -render-worker /path/to/blitz-worker-or-container-wrapper \
  -usb-worker /path/to/epaperstream \
  -serial /dev/cu.usbmodemEXAMPLE \
  -trusted-html \
  -debounce-interval 2s -max-wait 5s
```

Paths are placeholders. Use an operator-generated mode-0600 token file with at
least 32 bytes, separate from future device keys. Token, HTML and bitmap data
are not passed as worker argv. `-render-worker` selects this mode; USB worker,
serial and explicit `-trusted-html` are required. Enrollment/state files belong
to legacy mode, not this in-memory prototype.

Default bind: `127.0.0.1:8443`; explicit `[::1]:8443` is accepted. Non-loopback
binds reject. Existing TLS 1.3 and HTTP header/read/write/idle limits remain.

`-full-interval` defaults to 180s and cannot be lower for this EPS1 experiment.
The pump waits that interval at startup because the previous physical refresh
time is unknown, then after each successful delivery. Pending submissions may
be replaced during cooldown. This conservative prototype rule does not replace
the agreed future configurable policy or separate 600s maintenance requirement.
There is no startup image or automatic maintenance update in this slice.

## API

All routes require `Authorization: Bearer …` over HTTPS.

| Request | Result |
| --- | --- |
| `GET /v2/screen` | Complete HTML and strong ETag; initially empty revision zero. CSP sandbox prevents browser execution. |
| `PUT /v2/screen`, `Content-Type: text/html; charset=utf-8`, `If-Match: <ETag>` | Whole-scene replacement; `202 {"revision":N}` means queued, not displayed. |
| `GET /v2/screen/status` | `current`, `in_flight`, `confirmed`; no ETag on changing status. |

GET the tag before replacement. It contains a fresh random process epoch and
revision: a pre-restart tag cannot authorize a new process's revision. Require
one exact strong tag; wildcard/weak/list values cannot bypass the base check.
Missing base: 428; stale base: 412; oversized body: 413; unsupported type,
charset or compression: 415; invalid UTF-8: 400; large headers: 431; rate: 429.

Concurrent writes serialize; stale writers reject. This does not merge DOM
edits, subtrees or independent HTML snapshots. Fetch/rebase after conflicts.
Repeating PUT with its old tag returns 412, not another physical refresh.

`confirmed` means matching EPS1 terminal Commit success, not visible proof.
Renderer failure or transport ambiguity stops pump and service. Ambiguity
clears the confirmed baseline; another pump/request cannot silently retry it.
Explicit restart/full resynchronization is required, with startup cooldown.
In-memory content is not restored after restart. Device epoch/resync and a
confirmed pixel baseline for diff remain pending.

## Ownership, resources and corrections

The live API and pump sample elapsed monotonic time inside the queue mutex:
concurrent callers cannot submit earlier sampled clocks late. Rendering runs
outside the lock; a superseded result is discarded. Pump ownership is atomic
at **Screen** level. A review caught the earlier per-pump guard: two instances
could alternate leases, bypass cooldown or continue after another failed.
The distinct-pump regression test now guards the corrected ownership.

`frameio` centralizes BZM1 local process pipes: eight-byte header, bounded
geometry, tight MSB-first black=1 pixels, zero unused row bits and no trailing
data. Decode borrows an already bounded worker result; sending also borrows
frame pixels, without another full-frame copy. No Pico framebuffer is added.
BZM1 is not a network protocol and has no authentication or revision.

`streamusb.Sender` supervises one direct `epaperstream --raw` child per delivery
with a 180s context deadline and 1s pipe WaitDelay. Raw mode does not spawn
Blitz. Exit zero follows terminal success and DTR/port cleanup. Missing child,
nonzero exit, timeout or cleanup failure is ambiguous. Child diagnostics are
bounded to 1024 bytes; no token/HTML is passed in argv. Process startup per
full update is a deliberate isolation cost, not a measured energy improvement.

**Correction:** serial Close is best-effort for blocked writes. Pinned
go.bug.st/serial v1.8.0 explicitly wakes pending Unix reads on Close, but Write
uses a blocking syscall. The old close timer did not prove a bounded write.
Manager child-process supervision is the additional guard. Cable/kernel fault
acceptance remains separate. The firmware itself was not changed.

## Security and remaining work

Loopback/trusted-author restrictions remain until exact Blitz assets, URL
handling, parser/decode resource and OS-isolation gates pass. Do not publicly
reverse-proxy this prototype. TLS/token alone does not make arbitrary HTML safe.
EPS1 is still unauthenticated and never receives a network listener. WPA3,
new encrypted transport, USB/Wi-Fi arbitration, partial commands, image asset
policy and last-full-refresh corner are not implemented by this slice.
Secure public access remains the production goal, not a removed requirement.

## Evidence

- Focused race tests pass: authorization, concurrent/stale/pre-restart writes,
  unknown-length oversized input, type/charset/encoding, errors and budgets.
- TLS → API → pump → real EPS1 Link/Receiver test uses an in-memory sink and
  verifies one commit with both expected planes. It is not physical evidence.
- Subprocess tests exercise BZM1, literal argv, failure and cancellation; raw
  CLI tests use the real EPS1 codec/receiver. No real USB port in these tests.
- Live Linux manager in the Dev Container, temporary certificate: unauthenticated
  401, PUT 202, stale PUT 412, GET byte-identical HTML, status
  `current=1,in_flight=0,confirmed=0`, clean SIGTERM. Stopped before cooldown;
  deliberately configured no USB device. This is API runtime acceptance only.
- macOS arm64 manager and USB client cross-built inside the container. At this
  software checkpoint physical acceptance was pending; the subsequent S5 test
  below records protocol success and separate user-confirmed visible output.
- Independent review found and then rechecked the cross-pump fix; no required
  findings remained. Reviewer did not independently execute tests.

Final task gate: PASS, 17s; zero changed-file lint issues; changed executable
coverage 92.4%, total 89.8% against the 75.0% ratchet. Focused race tests and
live TLS smoke repeated after final source changes. Existing USB/Wi-Fi TinyGo
builds pass as regression checks; this does not accept the new Wi-Fi path.

Artifacts under ignored `build/stream/`, SHA-256:

- `epaper-manager-macos-arm64`:
  `303f1d66f777a74fe0cdee97df7ccdb0821a9c0e1fd5a108b4c61babd26603d7`
- `epaperstream-macos-arm64`:
  `0f4b1d3a801627583e823f441f4cfe2ef2e570637c0152fcde1710575d26e388`
- Unchanged `stream-device.uf2`:
  `a17ae2a2116c20f7af09d615c20ca7b006d42cb93d1a5240ff2c7eaa9a92185f`

Permission for the macOS loopback manager and one image was granted on
2026-09-06; protocol evidence and the subsequent visible confirmation are
recorded below. No BOOTSEL/reflash was required.

## Direct macOS manager → Pico test S5 — 2026-09-06

User authorized one upload. Matched existing container and artifact hashes;
serial node `/dev/cu.usbmodem2101` present. Native manager bound only
`127.0.0.1:19443`, using direct native `epaperstream` and the existing Blitz
container wrapper. Firmware, wiring, switches and 180s cadence were unchanged.

First GET failed before HTTP with native curl LibreSSL/3.3.6 and the Ed25519
smoke certificate. Server reported `peer doesn't support any of the
certificate's signature algorithms`. No scene or USB data was submitted.
Changed only the test certificate to RSA 2048, retaining TLS 1.3 and explicit
CA verification. The same client then completed GET/conditional PUT successfully.
Use an actual client handshake as the compatibility guard; never use `-k` or
a TLS downgrade to hide this error. No system trust store/tool changes made.

Sent exactly one full scene, ignored fixture `build/stream/acceptance-5.html`,
heading `MANAGER S5`, lower-right `ТЕСТ S5 · 06.09.2026`. The real Blitz preview
was inspected before submission. This badge is a fixed test label, not the
future last-full-refresh timestamp feature.

Measurements, Europe/Kyiv:

- Around 00:39:53: PUT returned `{"revision":1}`.
- During startup cooldown: `confirmed=0,current=1,in_flight=0`.
- By 00:43:25: `confirmed=1,current=1,in_flight=0`.

This protocol result confirms HTTPS → queue → actual Blitz worker → supervised
native USB worker → matching terminal EPS1 Commit success; alone it does not
prove visible pixels. No retry, second update, reflash or agent-triggered Pico reset.
Manager then stopped cleanly with SIGINT, exit 0; no background listener left.

**Visible S5 acceptance:** the user subsequently replied «так» to seeing
`MANAGER S5` and `ТЕСТ S5 · 06.09.2026` on the physical panel. S5 supersedes S4
as the latest visible checkpoint and accepts this one-scene localhost manager
→ Blitz → USB → Pico → panel flow. Full Wi-Fi/partial/energy acceptance is
not established by this test.

## Image acceptance S6 — 2026-09-06, protocol passed; visible acceptance failed

After checked BZR4 identity and Go static-content admission, prepare one new
scene `testdata/stream-image-s6.html`: heading `ЗОБРАЖЕННЯ S6`, two copies of one
synthetic embedded PNG and `ТЕСТ S6 · 06.09.2026` in the corner. The corner is a
fixed test label, not the last-full-refresh feature. The authored body explicitly
fills the viewport, following S5's positioned-body pattern; automatic-height
body corner behavior remains subject to the known positioning limitation.

`build/stream/s6/preview.png` passed actual Go → rebuilt Blitz rendering and
visual inspection, including both images and the readable corner. New macOS
arm64 binaries were cross-built **inside the existing Dev Container**; previous
S5 artifacts were preserved. SHA-256:

- Manager: `cd9615334ff6748388890cb462844444e0335bc98027a82f3bb6a40260e3c02c`
- USB worker: `12fc2cba747cc1a5f704f5e213d9c43f85a509c9d9e7563c3c22635d9dfc5a4c`
- Existing streaming UF2: `a17ae2a2116c20f7af09d615c20ca7b006d42cb93d1a5240ff2c7eaa9a92185f`

The prepared physical step required explicit permission to launch the native macOS
loopback manager and direct USB worker, plus a connected Pico. Do not execute
host tools under the assumption that container permission covers them. Recheck
the serial device, container wrapper and certificate without exposing secrets;
submit exactly this one scene over verified localhost TLS, retain the 180s
startup/cadence guard, then stop the manager after terminal success/failure.
No BOOTSEL, reflash, rewiring or automatic retry. Ask for separate visible
confirmation of heading, both image copies and corner only after terminal
success. S5 stays the last accepted physical checkpoint until then.

**Executed after user approval:** matched all three artifact hashes and the
running container; `/dev/cu.usbmodem2101` exists. Existing RSA certificate was
valid with SAN 127.0.0.1, token/key permissions 0600. Native manager started at
approximately 16:39:30 UTC on `127.0.0.1:19443`, verified TLS GET/conditional PUT
submitted one S6 scene by 16:40:09. Initial status was
`current=1,in_flight=0,confirmed=0`; startup cooldown remained 180 seconds.

At 16:42:59 UTC (19:42:59 Europe/Kyiv), status returned
`current=1,in_flight=0,confirmed=1`. Manager stopped immediately afterwards with
SIGINT, exit 0. No extra submit, retry, reflash, reset or container/USB attachment
change. This establishes terminal protocol success, not visible image proof.
User replied “ні, s5”: the panel still shows S5. S6 visible acceptance failed;
S5 remains the last accepted physical image. No additional send or reflash was
performed during the following read-only investigation.

Code inspection does not support a simple reused-ID explanation: each USB send
uses a fresh random epoch, and both passes are checked against the frame digest
before Commit. Focused streamwire/streamrx/streamusb/epaperstream tests pass.
The remaining observability gap is physical execution: `panel.waitIdle` accepts
an already-high BUSY input, and streaming firmware does not export panel events.
Thus terminal success cannot prove the controller actually refreshed the panel.
This is a diagnostic limitation, not proof of a wiring or controller fault.

### Subsequent isolation — captured S6 replay passed

After an unsuccessful S4 HTML resend, a bitmap-only half-black/half-white raw
control passed visible acceptance. Then the user authorized capture, inspection
and direct replay of the rendered S6 image. The existing Go manager Screen and
checked BZR4 worker produced `build/stream/s6/replay.bzm` (48,008 bytes), SHA-256
`85255ce0e5e2849b029b30885aeadea36eb6de49cd46445c857a9db193468011`.
The PNG decoded from those exact bytes matches the earlier S6 preview checksum.

One `epaperstream --raw` invocation sent that immutable file without running
the renderer or HTTPS manager during USB. Same S6 client and firmware, no
rewiring/reset or guard change. By 17:05:31 UTC it exited 0, `stream=committed`.
The user then confirmed «є зображенння»: the captured S6 replay is physically
accepted. Keep the BZM1 and PNG unchanged as reproducible control artifacts.

This establishes render-to-file and raw delivery for S6, not why prior live
delivery failed. Do not declare a manager/renderer or wiring defect fixed.
Earlier live-manager failure remains evidence, and S5 remains the last accepted
live HTTPS-manager delivery. No additional send followed this confirmation.

Sources checked 2026-09-05:

- [HTTP If-Match](https://www.rfc-editor.org/rfc/rfc9110.html#name-if-match)
- [Go MaxBytesReader](https://pkg.go.dev/net/http#MaxBytesReader)
- [Go process supervision](https://pkg.go.dev/os/exec#CommandContext)
- [Pinned serial implementation](https://github.com/bugst/go-serial/blob/v1.8.0/serial_unix.go)
