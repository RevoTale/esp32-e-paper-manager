# Verification record

## Latest checkpoint — 2026-09-06

- Captured S6 bitmap replay PASS: user confirmed the image after direct raw
  upload of the inspected, immutable BZM1. Its PNG checksum equals the earlier
  S6 preview; file SHA-256 is
  `85255ce0e5e2849b029b30885aeadea36eb6de49cd46445c857a9db193468011`.
  Same raw client and firmware as the successful control, no rewiring/reset.
  Renderer output and raw delivery work for this scene; earlier live S6/S4
  failures remain unexplained, not repaired. Latest live HTTPS success is S5.
- Bitmap-only physical control PASS: user confirmed the half-black/half-white
  800x480 image after one direct raw upload. Existing S6 USB client and unchanged
  streaming firmware; no HTML/Blitz/manager, reset or wiring change. S4 resend
  and the earlier live S6 remain failed tests; their cause is not established. The
  latest full manager-path success remains S5. Exact frame/client hashes and
  evidence: [debugging history](../../docs/epaper-debugging-history.md).
- Static HTML admission now rejects active content before PNG decoding/worker
  startup in native and bitmap paths. Five tests pass; independent review found
  details/summary were initially missed, reproduced and corrected with active
  tests (also open dialogs). No engine patch or new ignore. This is a Go-only
  admission layer, not full profile or public-input isolation.
- Final task gate PASS 17s, zero new lint issues, 93.6% changed / 90.2% total
  coverage. New gate: 23/23 changed executable lines covered. Focused race tests
  pass. USB/Wi-Fi build sizes and Rust 131-pass/3-ignore state are unchanged.
  See [static HTML boundary](docs/static-html-admission.md).
- S6 image scene rendered via actual Go/BZR4/Blitz and was visually checked;
  heading, two PNG copies and fixed corner label are present. Fresh macOS arm64
  manager/USB binaries were prepared separately from S5. After explicit host
  permission, exactly one scene reached terminal `confirmed=1` at 16:42:59 UTC,
  then manager stopped cleanly with exit 0. No reflash/reset/USB reattachment;
  unchanged 180s startup guard. User subsequently reported S5 still visible:
  S6 physical acceptance FAILED despite protocol completion. Cause is not yet
  established; no automatic resend. [S6 evidence](docs/screen-manager-usb.md).

### Preceding checked source identity checkpoint

- Checked Go/Blitz CSS identity: BZR4 is emitted for every Go scene, including
  empty manifests. Source kind/order/ordinal/text must match the actual DOM
  before image injection/paint; mismatch returns no frame. Legacy probes remain
  compatible without identity guarantees. Native/fallback policy stays opt-in.
- Fourteen added Rust tests bring the suite to 131 active passes, zero failures
  and the same three approved engine ignores. Both actual Go image and repaired
  HTML previews pass and were visually inspected. Independent review found no
  required remaining defect. No source text is exposed by mismatch diagnostics.
- Formatting and all-target Clippy deny-warnings pass. Initial Go lint found
  complexity 11 in the expanded wire test; exact record comparison preserves
  every assertion with independent expected ordinals and resolves the finding.
  Final task gate PASS 16s, zero new lint issues, 93.5% changed / 90.2% total
  coverage. Focused worker/manager/diagnostic race tests pass. USB flash/RAM
  237724/105564 bytes; Wi-Fi 699804/112156, unchanged. Six pre-existing file-length
  debts remain reported. See [identity contract](docs/css-source-identity.md).
- No new dependency, ignore, firmware change, flash, USB transfer, commit or
  push. S5 remains the last accepted visible scene. Fallback authorization,
  computed bounds, full profile and public-input isolation remain open.

### Preceding declaration binding checkpoint

- Actual-DOM CSS bindings: fourteen new tests bring Rust to 117 active passes,
  zero failures and the same three approved engine ignores. Rule/inline source
  locations, specified UTF-8/escaped declaration spans, mixed ownership, exact
  shared budgets and source-free diagnostics are covered. Same-document
  analysis preserves ordinary cascade output and independently asserted pixels.
- Independent review found the source-count cap was checked after copying the
  next source. The pre-copy guard and active regression test fix this; no bound
  was relaxed. Two new-fixture corrections (untrimmed selector spans, supported
  background-color longhand) are preserved in the root debugging history.
- Formatting, all-target Clippy deny-warnings and tracked/untracked whitespace
  checks pass. Go task gate PASS 19s, zero new lint issues, 93.5% changed/90.2%
  total Go coverage. USB flash/RAM 237724/105564 bytes; Wi-Fi 699804/112156,
  unchanged. The six pre-existing file-length debts remain visible.
- See [CSS binding contract](docs/css-style-bindings.md). Local bindings remain
  opt-in with native-core validation for all targets. Checked Go/Blitz source
  identity, fallback authorization/extraction, computed bounds and public-input
  isolation remain open. BZR3, firmware, USB and physical S5 acceptance are
  unchanged. No dependency change, new ignore, flash, commit or push.

### Preceding target ownership checkpoint

- Actual CSS target ownership: seventeen new real-DOM/limit tests bring Rust
  to 103 active passes, zero failures and the same three approved engine ignores.
  The opt-in scope reuses Blitz matching, preserves distinct native/bitmap owners,
  handles parser repair/duplicate IDs and rejects malformed/over-budget analysis.
- Independent review found a missing root-parent guard before upstream parent
  traversal. Active RED→GREEN tests now cover root/type/ID and child-link failures.
  An extra fixture pins Blitz's fixed `NoQuirks` case behavior, with and without
  doctype. No browser-quirks conformance or engine repair is claimed.
- Formatting, all-target Clippy deny-warnings and diff whitespace checks pass.
  Go task gate PASS 16s, zero new lint issues, 93.5% changed/90.2% total Go
  coverage. USB flash/RAM 237724/105564 bytes; Wi-Fi 699804/112156, unchanged.
  The six pre-existing file-length debts remain visible.
- See [target ownership contract](docs/css-target-ownership.md). Whole-rule and
  inline-declaration integration, fallback policy/extraction, stable render-object
  identity and public-input isolation remain open. BZR3, firmware, USB and S5
  physical acceptance are unchanged. No dependency change, new ignore, flash,
  commit or push.

### Preceding native selector checkpoint

- Native selector admission: eight added tests bring Rust to 86 active passes,
  zero failures and the same three approved engine ignores. Tested source-token
  policy, parser normalization, escaped literal matching, source-free UTF-8
  spans/budgets, unchanged BZR3 and exact real-render cascade pixels.
- A real-render fixture confirms that a stylesheet inside a bitmap-marked
  subtree changes an outside box. This guards against inferring CSS scope from
  the marker; actual ownership/extraction was not implemented at that checkpoint. See
  [selector and fallback boundaries](docs/native-css-selectors.md).
- Formatting, all-target Clippy deny-warnings and `git diff --check` pass.
  Go task gate PASS 18s, zero new lint issues, 93.5% changed/90.2% total Go
  coverage. USB flash/RAM 237724/105564 bytes; Wi-Fi 699804/112156, unchanged.
  Existing six file-length debts remain reported.
- No native activation in BZR3, engine fix, dependency upgrade, flash, USB
  transfer, commit or push. Public-input and full-profile acceptance remain open;
  the last accepted visible streaming scene is still S5.

### Preceding native declaration checkpoint

- Isolated native declaration core: `csscheck::Validator::native()` reuses
  pinned Stylo property IDs/expansion and adds allowed-value and numeric limits.
  Twelve added tests bring Rust to 78 active passes, zero failures and the same
  three approved engine ignores. Real Blitz pixel output, UTF-8 truncations and
  unchanged grammar-only BZR3 behavior are covered. Final alias/HSL cases also
  pass in the focused suite; formatting and all-target Clippy deny-warnings pass.
- Go task gate at that checkpoint: PASS 16s, zero new lint issues, 93.5% changed coverage and
  90.2% total Go coverage. USB flash/RAM 237724/105564 bytes; Wi-Fi
  699804/112156, unchanged. Six existing file-length debts remain reported.
- Native mode is not enabled for manager submissions. Selector/fallback
  ownership, font/background assets, broad shorthand resets and computed bounds
  remain required before integration. Specified-value caps do not establish a
  public-input sandbox or complete CSS support. See
  [native declaration contract](docs/native-css-declarations.md).
- No firmware flash or physical acceptance in this increment; last accepted
  visible streaming scene remains S5. No dependency upgrade, commit or push.

### Preceding checked-CSS and engine-deferral checkpoint

- Checked manager CSS sources share resource budgets and are validated before
  Blitz DOM construction. BZR3/BZE1 process tests and real Go image preview pass.
- Rejected documents preserve the confirmed frame, expose source-free revision
  diagnostics and wait for corrected input. Fatal renderer/frame failures cannot
  be hidden by a concurrent new scene. Focused race tests and review pass.
- Go task gate after the deferral: PASS 17s, zero new lint issues,
  93.5% changed coverage, 90.2% total.
  USB flash/RAM 237724/105564 bytes; Wi-Fi 699804/112156, unchanged. Existing six
  file-length debts remain reported, not suppressed.
- Before the latest deferral, Rust `--no-fail-fast`: 66 passing tests, two
  approved image ignores and one failure exposing incorrect containing-block selection through a static
  wrapper. The user subsequently approved BLITZ-POS-001: ignore only this exact
  engine reproduction without altering its assertions or patching Blitz. Full
  CSS conformance remains unaccepted. See [reproduction](docs/blitz-positioning-gap.md).
- After the deferral: Rust `--no-fail-fast` passes with 66 active passes, exactly
  three approved ignores (two image, one positioning), zero failures. Formatting
  and all-target Clippy with warnings denied pass. The exact `--ignored --exact`
  positioning reproduction still fails as documented; it is not a repaired bug.
- No new firmware flash, USB transfer, commit, push or physical acceptance.
  The last accepted visible streaming scene remains S5, not this server-only work.

Earlier dated records below describe their own checkpoints, not the current
renderer acceptance state.

Date: 2026-08-30

## Automated checks

Inside the Debian Trixie devcontainer:

```text
go test ./...  PASS
go test -race ./... PASS
go vet ./...   PASS
```

Content-based decode tests use real PNG, JPEG, and GIF byte streams with
misleading filename extensions. Image dimensions and detected formats match.
The host client also verifies Go's permitted `Read(n > 0, err != nil)` case:
received protocol bytes are decoded before the pending read error is surfaced.
Typed device rejections retain their protocol code and next expected offset,
and the CLI-facing error string includes both diagnostic values.
USB cleanup tests prove DTR is deasserted before the serial port closes; close
still runs if DTR deassertion fails, and both errors remain inspectable.
TCP endpoint tests cover DNS, IPv4, bare/bracketed IPv6 and reject empty hosts,
malformed forms, service names, port zero, and ports above 65535 before dial.

TinyGo target: `pico2-w`, scheduler: `tasks`, panel SPI0 mode 0 at 4 MHz.

Protocol resource delta against a probe retaining the same 48,000-byte caller
frame:

```text
static_delta=400 bytes
stack_delta=0 bytes
RAM_cost=400 bytes (limit 2048)
flash_cost=4632 bytes (limit 32768)
protocol heap allocations reported=0
```

TinyGo 0.41.1 emitted the same two recursive system-runtime stack warnings for
both probes and no numeric stack maximum. Both linked the same 4096-byte C
stack; the raw reports are retained under `resource-reports/protocol/`.

Composed USB firmware:

```text
flash=23596 bytes
RAM=54116 bytes, including the sole 48000-byte frame
selected-package heap allocations reported=0
remote-epaper.uf2 sha256=32071d9d0707672e875435b656e9fa4bbac9bd5ddf7c0f6f3b710b4d79ef7f7a
epaperctl sha256=2c462baf72f0977aab3baf2fe5b4fca6d881045e800c0fae8a745e86e763757c
```

Cross-compiled static Raspberry Pi/Linux clients (Go 1.26.2, CGO disabled,
trimmed paths):

```text
linux/arm64  4.6 MiB  sha256=21224ad974a275ff93df61e47696a42f77a9c1d7da86ba7d4fb7b45c4b3a9591
linux/armv7  4.5 MiB  sha256=2883220cc68ee2cad941eec90fb8ba59a6d537d4b3839f24987f327710adfe43
```

Secure transport probe (HMAC-SHA-256, AES-256-GCM, and the durable epoch
journal linked for `pico2-w`, but no radio/listener):

```text
flash=120500 bytes
RAM=5880 bytes
secure-check.uf2 sha256=0025eafd9ba95c7fff66c376bc4849ae6667a545c006f6734c64da417d984551
```

Security fuzz smoke runs:

```text
FuzzOpenRejectsHostileEnvelopesWithoutMutation: 1,663,150 executions / 3 s
FuzzHandshakeRejectsHostileBytes: 3,105,667 executions / 3 s
```

Update 2026-08-31: the authorized Wi-Fi implementation and its two final
4096-byte flash journal blocks now build only with the `wifi` tag and valid
build-time configuration. The ordinary USB build still links the fail-closed
no-op. `go test ./...`, `go vet ./...`, `go mod verify`, shell syntax
validation, and a scripted TinyGo build passed:

```text
target=pico2-w scheduler=tasks tag=wifi
flash=568368 bytes
RAM=56768 bytes
```

TinyGo stack analysis reports only the known conservative recursion warnings
through `runtime.nilPanic` for `Reset_Handler`, `runWiFi`, the network poll task,
and `runtime.run$1`; it provides no numeric maximum. Physical WPA2 association,
authenticated TCP transfer, USB preemption, radio-on current, and reconnect
after access-point loss remain pending.

Panel acceptance artifacts (not yet physically accepted):

```text
panel-a.uf2       e8f925666ea0f5178d089cb51643ea97948c3faf1b76350fcbbfd3a45c74ce45
panel-b.uf2       a6a8b4c6e2010e879d5b3f4ea0742411f611257bd237a835d5dd4d15a1e36f01
panel-checker.uf2 1cff24aa2c7490dc167352f573747b896b34707fc7f1330d520b3902a17a3e4b
```

Live OrbStack/USB discovery:

```text
USB application device: 2E8A:000A
Linux driver: cdc_acm
sysfs tty: ttyACM0 (166:0)
epaperctl -list: /dev/ttyACM0 found
44-byte protocol Hello read: no bytes within 5 s
```

The missing `Hello` means the currently running Pico firmware is not yet proven
to be `03-remote-epaper`; no frame or panel command was sent during this check.
`picotool load -f -v remote-epaper.uf2` stopped before writing with `Unable to
locate reset interface on the device`; the next flash requires physical
BOOTSEL mode.

## Pending physical evidence

- Move HAT PWR from GP16/pin 21 to GP15/pin 20 while fully unpowered.
- Flash and visibly verify A, B, and checker artifacts with 180 seconds between
  refresh starts.
- Flash `remote-epaper.uf2`, open the CDC port, and complete 20 USB transfers.
- Record current in disconnected idle, DTR-attached idle, transfer, panel
  refresh, and deep-sleep states; also record transfer/refresh duration. No
  energy claim or polling optimization is made until those samples exist.

## 2026-09-01 Waveshare V2 TinyGo port

The panel implementation now follows the pinned official Waveshare
`EPD_7in5_V2.c` V3.0 sequence. The host tests verify reset timing, command and
payload order, project-to-Waveshare frame polarity, exact BUSY deadlines,
single-attempt behavior, detailed SPI/BUSY errors, bounded observer events,
safe PWR/CS cleanup, reentry rejection, and zero steady-state allocations.

Checks run inside the existing Dev Container with TinyGo 0.41.1, Go 1.26.2,
and LLVM 20.1.1:

```text
experiments/03-remote-epaper: go test ./... PASS
experiments/03-remote-epaper: go vet ./... PASS
remote-epaper pico2-w build:  flash=24116 bytes RAM=54116 bytes
panel-checker pico2-w build:  flash=18328 bytes RAM=53736 bytes

experiments/07-tinygo-waveshare-7in5-v2-diagnostic: go test ./... PASS
experiments/07-tinygo-waveshare-7in5-v2-diagnostic: go vet ./... PASS
diagnostic pico2-w build: flash=24160 bytes RAM=53784 bytes
```

Artifacts:

```text
remote-epaper.uf2  6e195c636c8285deade718b385427350a2816c027502b2265f7e7a457894faea
panel-checker.uf2  79d7255db1ec6b0faa17a8f064909f6ef8d2683261b2f10ea0d2a822511c6cf3
diagnostic.uf2     6888c05c69c553777e9011a20f60bdb04fadd1274060a24a376e827b6058db8f
```

Physical checkerboard output and its complete USB diagnostic log remain
pending; software verification cannot prove write-only SPI reached the HAT.

## 2026-09-02 V2-01 quality bootstrap

The separate `tools/go.mod` pins `golangci-lint` v2.12.2 and `covercheck`
v0.2.0. The firmware `go.mod` did not change. The local wrapper closes the
reviewed upstream gaps for uncommitted changes and `cmd/**` coverage.

```text
quality fast: PASS twice (1-2s)
quality task: PASS twice (3-4s)
quality full: PASS twice after final review (8-9s)
changed executable-line coverage: 91.2%
total statement coverage: 75.0% (baseline 75.0%)
pico2-w USB firmware: flash=24116 bytes RAM=54116 bytes
file-length negative/positive test: PASS
untracked-code lint negative test: PASS
task-duration negative test: PASS after the completed run exceeded a 0s probe
```

The six pre-existing oversized files and 50 baseline lint findings remain
reported debt. Changed whole files cannot retain that debt; no exclusions or
lint suppressions were added.

## 2026-09-02 V2-02 display contract and testkit

`display.Frame` borrows one caller-owned packed monochrome buffer. The
synchronous `display.Device` contract forbids retaining or mutating it after
`Refresh` returns. Capabilities validate logical size, color model, refresh
mode, alignment, and exact stride without importing the Waveshare driver.

```text
display package statement coverage: 95.5%
testkit package statement coverage: 100.0%
changed executable-line coverage: 92.4%
total statement coverage: 76.6% (baseline 75.0%)
quality task: PASS (3s)
display-contract-check pico2-w UF2: 13,312 bytes
```

The 128x64 contract test proves a second size without runtime or panel changes.
Frame mutation, digest, and equality checks allocate zero times in host tests.
This is target compile evidence, not physical panel acceptance.

## 2026-09-02 V2-03 update contract

The versioned update envelope, borrowed document view, fixed display timestamp,
timezone value, content identity, result validation, typed diagnostics, and
independent static golden vector pass unit, corruption, malformed, and
compatibility tests.

```text
document package statement coverage: 100.0%
update package statement coverage: 93.5%
changed executable-line coverage: 91.1%
total statement coverage: 78.5% (baseline 75.0%)
quality task: PASS (4s)
update-contract-check: flash=79032 bytes RAM=41012 bytes UF2=158208 bytes
```

CRC-32 is only corruption detection. Network authentication/encryption remains
a later acceptance item. See `docs/update-contract-v1.md`.

## 2026-09-02 V2-04 tokenizer resource spike

The iterative tokenizer borrows its 32 KiB input, returns token views, applies
a forward-consumption work limit, rejects malformed/unsupported syntax, and
uses zero host heap allocations for the adversarial maximum-size stream.

```text
full device baseline: flash=24116 bytes static RAM=54116 bytes
full device + 32 KiB input: flash=24248 bytes static RAM=86900 bytes
full device + input + tokenizer: flash=26320 bytes static RAM=86924 bytes
tokenizer full-device delta: flash=2072 bytes static RAM=24 bytes
isolated tokenizer delta: flash=2136 bytes static RAM=24 bytes
```

TinyGo reports identical recursive runtime roots instead of a finite maximum
stack. This limitation is retained in raw reports. See
`docs/resource-budget.md`; no physical HTML rendering is claimed.

## 2026-09-02 V2-05 bounded HTML profile

The iterative parser accepts the documented static dashboard tags, quoted
allowlisted attributes, six named entities, comments, and HTML doctype. It
rejects implicit/mismatched markup, active/external features, unsupported
entities, and every structural/content limit before framebuffer mutation.

```text
sanitized Glance fixture: PASS
maximum-input parser steady-state host allocations: 0
full device + input + parser nodes: flash=30164 bytes static RAM=97268 bytes
parser differential: flash=5916 bytes static RAM=10368 bytes
```

See `docs/html-profile-v1.md`. Transport fragmentation is validated before the
complete borrowed HTML buffer reaches this parser; physical rendering remains
pending.

## 2026-09-02 V2-06 bounded CSS profile

Simple element/class/ID selectors, specificity/source order, inline priority,
bounded properties and units, tag defaults, inheritance, and hidden metadata
elements pass the accepted/rejected value and global-declaration matrices.

```text
css package statement coverage before final gate: 96.5%
full device + parser + CSS: flash=35972 bytes static RAM=102972 bytes
parser/CSS differential: flash=11724 bytes static RAM=16072 bytes
```

See `docs/css-profile-v1.md`. This verifies parsing/resolution and target
resource linkage, not layout, raster output, or physical display behavior.

## 2026-09-02 V2 final software gate

The final software path includes bounded HTML/CSS/layout/raster, USB updates
and provisioning, WPA3-only outbound networking, TLS 1.3 manager API,
HMAC-SHA-256/AES-256-GCM device sessions, persistent two-phase delivery status,
USB priority, reconnect/backoff, and the Waveshare full-refresh adapter.

```text
quality task: PASS (20s)
quality full: PASS twice (70s, 54s)
lint issues: 0
changed executable-line coverage: 90.0%
total statement coverage: 88.2% (baseline 75.0%)
USB firmware: flash=237900 bytes static RAM=105564 bytes
WPA3 firmware: flash=700320 bytes static RAM=112156 bytes
complete render probe static RAM=156580 bytes (limit 160000)
```

Final artifacts:

```text
remote-epaper-usb.uf2  sha256=7866b6751aaa3580ee17207ee452df79493d8ad6f8c20b6555309e05ff34fc6d
remote-epaper-wifi.uf2 sha256=c1d683a55f19f0b801676b33e7a2c65668826ed3faf8ec77cd62aa7926e6d06a
epaperhtml              sha256=e214ebb78ded1a8a2cbec73cff1d4b583c558544ef5eccda5f70cb46f4f89051
epaperprovision         sha256=c0c0d32381541a8b2b494db011a2a8602f915717a9545966b3f5159060aaa8fa
epaper-manager          sha256=ed9b9dd5b42947201ba6b22db190fa2a6b7064f5ff6980d6c2db36fb1ebda2fb
```

No Bluetooth or insecure Wi-Fi authentication symbol was found in the final
Wi-Fi UF2 string audit. Target IPv6 remains unavailable and fails closed.
Physical USB, WPA3, panel, router-loss, current, timing, and battery acceptance
remain separate mandatory evidence.

## 2026-09-02 post-flash USB observability

- Generic CDC enumerated on macOS as TinyGo `2E8A:000A`.
- The new diagnostic handshake returned `usb=ready provision_state=0`.
- `testdata/post-flash-success.html` first returned `status=1` (accepted and
  queued). The visible `Firmware installed` page was later shown to be a
  retained old frame, so this was not physical acceptance of that transfer.
- The sanitized dashboard reached `usb=ready` but the update stage exceeded the
  20-second bound. The protocol was changed to bounded 64-byte frames with
  per-frame ACK because TinyGo CDC has a 512-byte RX ring. The host now waits
  for terminal `refreshed` or `rejected`, not only `accepted`.
- `status=refreshed` still did not change the panel. A checker reported
  `panel refresh complete; HAT power is off`, but sampled `BUSY` three times
  with zero active-low observations. This localized the remaining problem to
  the physical controller/power path rather than HTML or USB framing.

## 2026-09-02 intermittent HAT PWR connector

- HAT `VCC` measured approximately `3.2 V`, so its logic rail was present.
- A dedicated probe held `RST=0` and alternated `GP15/PWR` every five seconds.
- During HIGH, the Pico side and female jumper contact measured approximately
  `3.2 V`; HAT `PWR` measured approximately `0.01 V`.
- Probe pressure and connector movement could temporarily restore continuity,
  explaining inconsistent resistance measurements.
- Adding one more female-male jumper to the PWR connection restored operation.
  This physically confirms an intermittent female Dupont contact in the PWR
  path. It does not implicate the HTML renderer, USB protocol, SPI driver, HAT,
  or panel.
- Regression rule: verify dynamic voltage at both endpoints of a suspect wire;
  continuity alone is insufficient. A visible unique frame remains the final
  physical acceptance criterion.

## 2026-09-02 USB HTML end-to-end physical acceptance

After the PWR jumper workaround, the main USB firmware was restored through
the verified TinyGo CDC 1200-baud bootloader transition and direct macOS UF2
copy.

```text
firmware sha256=8d9430b8feb76a3811e6308e9b14c1ca1cfa325dbe906348318cd64c1fa676aa
usb=ready provision_state=0
usb=transmitted
status=accepted id=f9baf5937ce8fe16486357f4d3ad88bc
status=refreshed id=f9baf5937ce8fe16486357f4d3ad88bc
content sha256=9a112ec81ed990be1b8b3c36d4d628859b99a3dbf5b17c1ccdfe7a80b266326a
visible result=USB UPDATE confirmed by user
```

This physically accepts one complete USB HTML path through firmware boot, CDC,
framed transfer, local render, Waveshare driver refresh, and visible pixels.
It supersedes the earlier false acceptance based on the retained
`Firmware installed` frame. It does not yet accept the 20-update endurance
run, Wi-Fi/WPA3 operation, reconnect behavior, current draw, or battery life.

## 2026-09-02 bounded rapid-refresh demonstration

- Production behavior was first confirmed: a second request after ten seconds
  returned `accepted` but did not refresh because the 180-second panel guard
  remained active.
- An explicit `rapid_demo` build selected a tested 10-second minimum while the
  default build retained 180 seconds.
- Three distinct HTML documents (`STARTING`, `PROCESSING`, `COMPLETE`) each
  reached terminal `refreshed` after ten-second waits.
- The device was immediately restored to production artifact SHA-256
  `8d9430b8feb76a3811e6308e9b14c1ca1cfa325dbe906348318cd64c1fa676aa`.
- This is evidence for bounded queue scheduling and repeated USB rendering. It
  is not approval for continuous ten-second full refreshes.
- Intermediate `accepted` hashes were repeatedly corrupted near their tail;
  terminal `refreshed` hashes were stable. This diagnostic defect remains open.

```text
quality task: PASS (18s)
quality full: PASS (56s)
lint issues: 0
changed executable-line coverage: 90.3%
total statement coverage: 88.2%
epaperhtml-darwin-arm64 sha256=8ce766643416508442571fbf90d0dfd7a976eb117ff0c808d80f613becc805f6
```

## 2026-09-06 manager viewport/CSS characterization

Ten new real-Blitz pixel fixtures pass; full Rust suite is 26 pass, 2 existing
approved image-layout ignores, 0 fail. Formatting and all-target Clippy pass.
Real Go preview visually confirms readable Ukrainian text, inline alignment,
boxes and viewport-relative placement. No production code, firmware or hardware
change; this is not a new physical checkpoint. Exact fixture scope, commands,
review and remaining profile gates: [`blitz-viewport-css.md`](docs/blitz-viewport-css.md).
