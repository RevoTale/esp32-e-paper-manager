# Pi 5 service tasks

Status: software complete and tested; target hardware/systemd acceptance pending.
Run project commands through
the existing Dev Container. Command paths below are relative to
`/workspaces/pico-sandbox/experiments/03-remote-epaper`.

## P1 — exact hardware source contract

- [x] Retrieve pinned Waveshare commit and compare V4 C/Python implementations.
- [ ] Inspect rendered electrical tables and Rev2.1 schematic.
- [ ] Record reset/BUSY/SPI/sleep behavior, supply limits and attribution;
  distinguish board-level values from bare-panel absolute maxima.

Verify: immutable commit/file references and reviewed source-to-behavior table.
Dependencies: none. Scope: small; investigation note and panel-v4 spec.

## P2 — one portable full update

- [x] Implement `display.Device` in `panelv4` with narrow error-returning I/O.
- [x] Prove one validated 4000-byte canonical frame follows the exact full
  lifecycle, polarity conversion and six padding-bit policy without mutation.
- [x] Invalid frame/mode and overlapping calls fail without extra hardware I/O.

Verify: `go test -race -cover ./panelv4 ./display`; red/green bus transcript.
Dependencies: P1 command-source verification complete; electrical review still
blocks physical wiring, not the portable fake-I/O tests.

## P3 — faults and portable build

- [x] Test every I/O failure, real elapsed-time deadline, BUSY transitions,
  incomplete upload and next-update recovery with fake time.
- [x] Never send activation after failed upload or sleep while stuck BUSY.
- [x] Add a reproducible TinyGo probe for the portable driver; run existing
  renderer/display regressions without modifying old firmware behavior.

Verify: focused fault tests, TinyGo artifact, unchanged old-target build gates.
Dependencies: P2. Scope: medium, driver/fault tests/build-probe wiring; split
into two commits of work if more than five source files are needed (do not
actually commit without user request).

## Checkpoint A

- [ ] Source contract, successful trace, fault traces and TinyGo proof reviewed.
- [x] No physical output claim; no dependency on unverified GPIO configuration.

## P4 — Linux hardware adapter

- [x] Pin reviewed character-device library; verify RP1 chip/line identity
  using an explicit configurable path and reject occupied lines.
- [x] Implement error propagation and rollback; exactly one CS owner.
- [ ] Supply reviewed Pi 5 TX-only SPI overlay and final eight-wire table,
  excluding all documented Pironman pins; validate against a target kernel.

Verify: `go test -race -cover ./linuxpanel`, overlay compile/application against
matching base DTB, documented runtime ownership check before hardware use.
Dependencies: P3. Scope: split into P4a library/adapter (up to five files) and
P4b overlay/wiring/tests (up to four files). Target inspection may need user.

## P5 — real input to a fake device

- [x] Reuse engine for bounded HTML and PNG; validate decoding before I/O.
- [x] Test landscape rotation, fit, alpha, odd stride and metadata corner.
- [x] Prove unchanged suppression, refresh floor and failure invalidation.

Verify: `go test -race -cover ./localdisplay`; real fixture pixels and fake
display lifecycle, not a mocked renderer only.
Dependencies: P3. Scope: P5a conversion and P5b scheduling, each up to five files.

## P6 — Unix HTTP transport

- [x] Implement authenticated status/frame handlers with bounded admission,
  stable error codes, source-free warnings and no unbounded queue.
- [x] Test parallel clients, bad token/MIME/body, disconnect before/after
  activation, status recovery and socket directory permissions.

Verify: real Unix-socket HTTP tests via `go test -race -cover ./localdisplay`.
Dependencies: P5. Scope: medium, up to five handler/transport/test files.

## Checkpoint B

- [x] Real HTML and PNG Unix POSTs reach expected fake-device frames; reconnect works.
- [x] Faults remain observable; no endpoint claims visible acceptance.

## P7 — Debian executable and Docker client

- [x] Compose `cmd/epaper-local`, credentials, signal handling and bounded
  shutdown. No display update merely on process startup.
- [x] Add hardened systemd unit and least-privilege socket/device permissions.
- [x] Add Docker bind/group/credential example without privileged or host
  networking; test reconnect after service/socket replacement.

Verify: `go test -race ./cmd/epaper-local`; `CGO_ENABLED=0 GOOS=linux
GOARCH=arm64 go build -o build/epaper-local ./cmd/epaper-local`;
`systemd-analyze verify` in a suitable environment and authorized Docker test.
Dependencies: P4, P6. Scope: P7a executable and P7b deployment, each <=5 files.

## P8 — final acceptance

- [ ] Run `sh scripts/quality.sh task` and the full relevant regression suite;
  review code, simplify without behavior changes, re-run affected gates.
- [ ] On approved Pi target, inspect kernel/pin ownership, install configuration
  and submit distinct HTML and PNG from Docker. User confirms actual pixels.
- [ ] Verify sleep/wake, restart/cooldown and Pironman RGB/OLED/fans; record
  binary hash, wiring, source versions, logs and optical evidence.

Dependencies: P7. Scope: acceptance docs and any separately tested fixes.
Do not check off physical gates based on HTTP success or BUSY alone.

## Software checkpoint — 2026-09-08

- New `panelv4` package only; old Pico drivers/entry points unchanged.
- RED observed before implementation; full command/register sequence and
  row polarity/padding then passed against deterministic fake I/O.
- Every synchronous I/O failure position tested, including retry by a new
  explicit refresh, upload interruption and deep-sleep failure. Fake clock
  tests cover initially HIGH, HIGH-to-LOW, post-activation stuck HIGH and
  elapsed BUSY read time. Concurrent calls reject rather than wait.
- `go test -race -cover ./panelv4 ./display`: pass; panelv4 coverage 100%.
- `tinygo test -c -target=pico2-w -scheduler=tasks ... ./panelv4`: pass, no
  flashing. Probe added to the existing `quality.sh` build/task/full gates.
- `sh scripts/quality.sh task`: PASS in 27 seconds; changed coverage 100%,
  total 91.8%, lint clean and all three pre-existing firmware builds pass.
  Existing file-length debt was reported, not changed or newly suppressed.
- Target kernel supplied: `6.18.39+rpt-rpi-2712`; upstream 6.18 SPI5 still
  requires avoiding stock GPIO12/13. Next input: current SPI/GPIO device nodes
  and permissions. No target
  service installation, GPIO claiming, wiring or optical acceptance yet.

## Overlay checkpoint — 2026-09-08

- User supplied device list: gpiochip0 and gpiochip10..13; gpiochip4 symlinks
  gpiochip0. GPIO nodes root:gpio 0660; SPI nodes root:spi 0660. Existing SPI
  devices are 0.0, 0.1 and 10.0. This does not establish GPIO chip identity or
  line ownership; no SPI5 device reported.
- Added candidate SPI5 TX-only overlay, GPIO14/15 plus kernel-owned CS16,
  and mandatory dtc/fdtoverlay/fdtget tests under `deploy/pi5`.
- RED: missing overlay compilation failed. GREEN: compilation, fixture merge
  and resolved pin/CS/SPI0/SPI10 assertions passed. Focused lint: 0 issues.
- Persisted `device-tree-compiler` in Dockerfile and installed it in the
  already-running container; no rebuild or host-system install performed.
- Real installed DTB merge and active overlays remain unverified. P4b is not
  complete and no physical pin table or deployment approval is implied.

## Adapter checkpoint — 2026-09-08

- User explicitly deferred target DTB merge/manual acceptance until the service
  is ready. Continue software P5..P7; do not repeatedly ask for the DTB.
- Reported symbols: spi5 `/axi/pcie@1000120000/rp1/spi@64000`,
  gpio `/axi/pcie@1000120000/rp1/gpio@d0000`. GPIO14..16 reported `none`.
  These are a snapshot, not proof of future availability or physical wiring.
- `linuxpanel`: GPIO identity/ownership, SPI readback, short-write failure,
  rollback, active-high requests and idempotent close implemented.
- Focused race tests passed with 93.2% statement coverage. No real GPIO or SPI
  device was opened in tests; native ioctl failure uses a temporary regular file.
- No homelab installation or hardware output claim. Pico code unchanged.
- Full `quality.sh task` passed in 31s: changed coverage 97.2%, total 91.9%,
  lint clean; all retained firmware build gates passed. No constraints relaxed.

## Service completion — 2026-09-08

- P5/P6/P7 software is implemented in `localdisplay`, `cmd/epaper-local` and
  `deploy/pi5`. Actual Unix HTML/PNG requests reach the expected fake pixels.
  Reconnect, cancellation before/after hardware and real-process SIGTERM pass.
- Independent review required better pre-hardware attempt status, real HTTP
  disconnect tests and complete operator docs; all findings were addressed
  and re-reviewed. Conservative dedup invalidation on invalid admitted requests
  remains intentional; no automatic retry or hidden partial refresh.
- Full quality gate passed in 27s: changed coverage 92.9%, total 91.9%; all old
  firmware builds and new CGO-free Linux ARM64 build pass. Full race/vet pass.
- Pinned govulncheck v1.1.4 reported no vulnerabilities for the service.
- `systemd-analyze` is unavailable in the Dev Container; unit policy tests pass,
  but the documented target syntax/sandbox check remains required.
- Operator guide: module `deploy/pi5/SERVICE.md`. P1/P4 real electrical/DTB
  acceptance and P8 visible output stay unchecked. No Pi deployment performed.
