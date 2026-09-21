# Remote HTML e-paper v2 tasks

Status: approved and in progress. Checkboxes are evidence-based; implementation
alone does not complete a physical acceptance item.

## V2-01: Quality bootstrap

- [x] Add a separate pinned Go tools module without changing firmware `go.mod`.
- [x] Configure lint, 300-line file guard, changed coverage, total coverage, and
  TinyGo build/resource commands.
- [x] Provide fast (local), task-end (target <=90 s), and full gates.
- Acceptance: versions are pinned; existing oversized-file debt is reported and
  cannot grow; new code cannot bypass the agreed limits.
- Verify: run every gate twice in the Dev Container and compare results.
- Likely files: `tools/go.mod`, `tools/go.sum`, `.golangci.yml`, `scripts/quality.sh`.
- Dependencies: none.
- Evidence (2026-09-02): `fast`, `task`, and `full` pass in the Dev Container;
  task gate takes 4 seconds, changed coverage is 91.2%, total coverage is 75.0%,
  and the `pico2-w` build reports 24,116 flash bytes and 54,116 RAM bytes.

## V2-02: Display contract and testkit

- [x] Define monochrome dimensions, pixel/frame ownership, refresh, sleep, and
  capability contracts without importing a concrete panel.
- [x] Add reusable fake display, clock, and allocation-free frame assertions.
- Acceptance: a second display size is testable without panel/runtime changes.
- Verify: contract tests plus TinyGo compile probe.
- Likely files: `display/surface.go`, `testkit/display.go`,
  `display/surface_test.go`, `cmd/display-contract-check/main.go`.
- Dependencies: V2-01.
- Evidence (2026-09-02): contract/fake tests pass with 95.5% and 100% statement
  coverage in `display` and `testkit`; changed executable-line coverage is
  92.4%; the
  `display-contract-check` probe compiles for `pico2-w` as a 13,312-byte UF2;
  `quality task` passes in 4 seconds. No firmware dependency changed.

## V2-03: Document and update contracts

- [x] Define bounded document/style, update request/result, typed errors,
  idempotency, timestamp, and diagnostics.
- [x] Add versioned independent golden vectors.
- Acceptance: USB, manager, and device link can share the contract without
  importing parser, renderer, or panel internals.
- Verify: unit, golden, malformed, and compatibility tests.
- Likely files: `document/model.go`, `update/contract.go`,
  `update/codec.go`, `update/contract_test.go`.
- Dependencies: V2-01.
- Evidence (2026-09-02): host unit, static golden, corruption, malformed, and
  compatibility tests pass; `document` coverage is 100.0%, `update` coverage
  is 93.5%, changed coverage is 91.1%, and total coverage is 78.5%. The
  `update-contract-check` target probe compiles at 79,032 flash bytes and
  41,012 RAM bytes. See `docs/update-contract-v1.md`. This is a transport
  contract, not physical USB or network acceptance.

## V2-04: TinyGo parser resource spike

- [x] Prototype the minimum tokenizer/state model and compile it for `pico2-w`.
- [x] Record flash, RAM, stack, allocation, and worst-case-input measurements.
- Acceptance: measured headroom is sufficient for one framebuffer, runtime,
  Wi-Fi, and parser; otherwise revise the HTML profile before full coding.
- Verify: reproducible resource report and adversarial 32 KiB input.
- Likely files: `html/tokenizer_spike.go`, `cmd/html-resource-check/main.go`,
  `scripts/check-html-resources.sh`, `docs/resource-budget.md`.
- Dependencies: V2-01 and V2-03.
- Evidence (2026-09-02): the full-device 32 KiB input probe uses 86,924 static
  RAM bytes; tokenizer differential is 2,072 flash and 24 static RAM bytes.
  The isolated 32 KiB adversarial test allocates zero heap objects. TinyGo
  reports identical recursive runtime roots rather than a finite maximum
  stack; parser code itself is iterative. See `docs/resource-budget.md`.

## V2-05: Bounded HTML profile

- [x] Implement the documented complete-buffer tokenizer/parser and supported
  tags. Transport fragmentation is resolved before this borrowed-buffer stage.
- [x] Reject nesting, token, attribute, entity, and input-limit violations with
  stable typed diagnostics.
- Acceptance: sanitized Glance input parses deterministically with bounded work
  and no unbounded recursion or retained source copy.
- Verify: table, split-boundary, malformed, fuzz, and resource tests.
- Likely files: `html/tokenizer.go`, `html/parser.go`, `html/errors.go`,
  `html/parser_test.go`, `html/fuzz_test.go`.
- Dependencies: V2-03 and accepted V2-04 spike.
- Evidence (2026-09-02): the sanitized fixture and malformed/limit/fuzz seed
  tests pass with zero steady-state parser allocations. Full-device parser
  probe uses 30,164 flash and 97,268 static RAM bytes; its differential over
  the same input-buffer baseline is 5,916 flash and 10,368 RAM bytes. See
  `docs/html-profile-v1.md` and `docs/resource-budget.md`.

## V2-06: CSS profile and style resolution

- [x] Implement only accepted selectors, properties, units, defaults, cascade,
  and inheritance.
- [x] Reject unsupported syntax according to the versioned contract.
- Acceptance: supported output is deterministic and unsupported CSS cannot
  cause unbounded work or memory growth.
- Verify: property/selector matrices, malformed input, and golden styles.
- Likely files: `css/parser.go`, `css/style.go`, `css/errors.go`,
  `css/parser_test.go`, `css/golden_test.go`.
- Dependencies: V2-03 and V2-05.
- Evidence (2026-09-02): selector/property/value/limit/default/inheritance and
  inline/style-element cascade tests pass. The full-device parser/CSS probe is
  35,972 flash and 102,972 static RAM bytes; differential over the same input
  baseline is 11,724 flash and 16,072 RAM bytes. See
  `docs/css-profile-v1.md` and `docs/resource-budget.md`.

## V2-07: Layout and overflow

- [x] Implement display-aware boxes, wrapping, spacing, content priority, and
  the reserved bottom-right timestamp rectangle.
- [x] Return typed clip/reflow/overflow outcomes with element context.
- Acceptance: content never writes outside the surface and the timestamp region
  remains readable on all tested display sizes.
- Verify: layout tables and multi-size golden geometry tests.
- Likely files: `layout/engine.go`, `layout/overflow.go`,
  `layout/engine_test.go`, `layout/golden_test.go`.
- Dependencies: V2-02, V2-05, and V2-06.
- Evidence (2026-09-02): multi-size geometry and overflow tests pass with 98.1%
  statement coverage. Layout reserves an 88x16 timestamp rectangle and reports
  fit/wrapped/clipped/elided/hidden outcomes. See `docs/layout-raster-v1.md`.

## V2-08: Monochrome rasterizer

- [x] Render accepted text, borders, fills, and assets directly into one 1-bit
  framebuffer through `display-surface`.
- [x] Add deterministic font/assets and timestamp rendering.
- Acceptance: no hidden full-frame allocation and pixel output is independent
  of the concrete e-paper driver.
- Verify: pixel-exact goldens, bounds tests, allocations, and TinyGo probe.
- Likely files: `raster/raster.go`, `raster/text.go`, `raster/assets.go`,
  `raster/raster_test.go`, `raster/golden_test.go`.
- Dependencies: V2-02 and V2-07.
- Evidence (2026-09-02): pixel-golden and 128x64/320x200 zero-allocation tests
  pass with 95.2% raster coverage. The target probe uses 49,028 flash and
  105,340 static RAM bytes. HTML cannot paint into the firmware-owned timestamp
  rectangle. See `docs/layout-raster-v1.md` and `docs/resource-budget.md`.

## V2-09: Render pipeline

- [x] Join parse, style, layout, raster, hashing, timestamp, and diagnostics.
- [x] Enforce 32 KiB input, five-second target, and stall-only watchdog rules.
- Acceptance: sanitized Glance fixture produces a stable golden frame and useful
  errors identify the failed stage and limit.
- Verify: end-to-end fixture, boundary, cancellation, stall, and resource tests.
- Likely files: `render/pipeline.go`, `render/result.go`,
  `render/pipeline_test.go`, `render/golden_test.go`.
- Dependencies: V2-05 through V2-08.

## V2-10: USB-first device runtime

- [x] Adapt v1 USB/runtime to accept the v2 update contract and HTML payload.
- [x] Finish simulated partial input, backpressure, detach, retry, and USB-priority
  behavior without interrupting an active refresh.
- [ ] Complete 20 consecutive physical USB HTML updates and detach/retry matrix.
- Acceptance: corrupt/incomplete documents never refresh; USB deterministically
  preempts incomplete network work; retry is idempotent.
- Verify: simulated transport matrix and 20 consecutive physical USB updates.
- Likely files: `runtime/runtime.go`, `transport/usb.go`,
  `runtime/runtime_test.go`, `transport/usb_test.go`, `cmd/device/main.go`.
- Dependencies: V2-03 and V2-09.

## V2-11: Known-good panel adapter and physical USB slice

- [x] Put the existing verified Waveshare 7.5-inch sequence behind the new
  display contract without changing commands/timing.
- [ ] Send the sanitized fixture through USB and record frame/build hashes,
  BUSY timing, refresh result, and recovery behavior.
- Acceptance: expected HTML and timestamp are visibly rendered after cold boot
  and retry, with required spacing between full refreshes.
- Verify: host goldens plus signed physical acceptance checklist.
- Likely files: `panel/adapter.go`, `panel/adapter_test.go`,
  `docs/physical-acceptance.md`, `scripts/build-device.sh`.
- Dependencies: V2-02 and V2-10.

## V2-12: USB-only credential store

- [x] Implement atomic integrity-checked configuration, generation counters,
  rotation, recovery, corruption handling, and factory reset.
- [x] Keep secrets out of logs, update payloads, public APIs, and generic UF2.
- Acceptance: interrupted writes recover to the last valid generation; missing
  or invalid configuration keeps Wi-Fi disabled.
- Verify: storage fault matrix, redaction tests, and TinyGo resource probe.
- Likely files: `provision/config.go`, `provision/store.go`,
  `provision/store_test.go`, `provision/fault_test.go`.
- Dependencies: V2-01 and V2-03.

## V2-13: Provisioning host tool

- [x] Add USB commands for inspect, provision, rotate, erase, and diagnose.
- [x] Validate WPA3-only mode, manager address, Europe/Kyiv default, and secret
  input without persisting it in shell history or project files.
- Acceptance: one generic UF2 can be safely configured and recovered entirely
  through USB with explicit operator confirmation for erase.
- Verify: fake-device integration and clean-machine operator walkthrough.
- Likely files: `cmd/epaperctl/provision.go`, `cmd/epaperctl/config.go`,
  `cmd/epaperctl/provision_test.go`, `docs/provisioning.md`.
- Dependencies: V2-10 and V2-12.

## V2-14: Home manager API

- [x] Implement authenticated HTTPS submission, one bounded pending update,
  device status, idempotency, expiry, and retention policy.
- [x] Keep device keys isolated from user/API credentials.
- Acceptance: public clients can submit through HTTPS/VPN while no Pico inbound
  port or device key is exposed.
- Verify: API contract, authz, size, expiry, concurrency, and restart tests.
- Likely files: `manager/server.go`, `manager/store.go`,
  `manager/server_test.go`, `manager/store_test.go`, `cmd/epaper-manager/main.go`.
- Dependencies: V2-03.

## V2-15: Authenticated encrypted device link

- [x] Implement Pico-initiated sessions with HMAC-SHA-256 authentication,
  AES-256-GCM records, nonces/counters, replay rejection, and key separation.
- [x] Add bounded reconnect/backoff and USB arbitration.
- Acceptance: tamper, replay, wrong-key, counter-wrap, truncation, and power-loss
  cases fail closed without losing USB operation.
- Verify: independent vectors, negative matrix, integration, and TinyGo resource
  tests; security review before enabling radio.
- Likely files: `devicelink/session.go`, `devicelink/record.go`,
  `devicelink/session_test.go`, `devicelink/vectors_test.go`,
  `runtime/network.go`.
- Dependencies: V2-10, V2-12, and V2-14.

## V2-16: WPA3-only IPv4 acceptance

- [x] Configure CYW43439 code for WPA3-only association with no downgrade.
- [ ] Verify DNS, manager connection, reconnect/backoff, USB priority, cold boot,
  router loss, and repeated power cycles.
- Acceptance: WPA2/open/TKIP networks are rejected and radio remains disabled
  when secure configuration or platform support is absent.
- Verify: automated host checks plus physical network acceptance matrix.
- Likely files: `network/wifi.go`, `network/wifi_test.go`,
  `docs/network-acceptance.md`, `cmd/device/main.go`.
- Dependencies: V2-15 and confirmed TinyGo/CYW43439 WPA3 capability.

## V2-17: IPv6 and public-manager acceptance

- [ ] Add target IPv6 address/DNS/connection support without weakening IPv4;
  blocked by the embedded stack's missing complete SLAAC/router lifecycle.
- [x] Implement a public dual-stack HTTPS manager route while Pico remains outbound-only
  and the home router has no inbound Pico forwarding.
- Acceptance: both address families pass the same authenticated encrypted update,
  reconnect, USB-priority, and failure semantics.
- Verify: dual-stack integration tests and physical/public route checklist.
- Likely files: `network/address.go`, `network/address_test.go`,
  `devicelink/dial.go`, `devicelink/dial_test.go`, `docs/public-access.md`.
- Dependencies: V2-16.

## V2-18: Energy, longevity, and final acceptance

- [ ] Measure physical render/transfer/radio/BUSY/current behavior; automated
  flash/RAM/resource gates are complete. Record timing, refresh spacing,
  coalescing, sleep, and repeated recovery.
- [x] Complete software documentation and map requirements to automated or physical
  evidence.
- [x] Finish the final repeated full-gate/review pass; full gate passes twice
  with lint 0, changed coverage 90.0%, and total coverage 88.2%. Repeat review,
  fix, and simplification loops until all agreed gates pass without weakening
  suppressions.
- Acceptance: generic UF2, tools, manager, docs, and hardware evidence are
  reproducible; all carried v1 acceptance gaps are closed.
- Verify: full quality gate and signed capability/evidence matrix.
- Likely files: `docs/acceptance.md`, `docs/resource-budget.md`,
  `docs/operations.md`, `CAPABILITY-MAP-v2.md`.
- Dependencies: V2-01 through V2-17.

# Preserved v1 tasks

The incomplete v1 checklist follows unchanged. Its open acceptance items are
carried into the v2 tasks above and must not be silently discarded.

# Remote e-paper v1 tasks

## Task 1: Frame record codec

- [x] Implement exact v1 header/trailer encoding and role-aware validation.
- [x] Implement fragmented incremental decoding and bounded resynchronization.
- [x] Add independent golden vectors, split-boundary, corruption, and fuzz tests.
- Verify: `go test ./protocol -run 'Codec|Decoder|Golden|CRC'`.
- Files: `protocol/codec.go`, `protocol/codec_test.go`, `protocol/fuzz_test.go`.
- Dependencies: none.

## Task 2: Frame receiver

- [x] Implement session ownership, sequential chunks, CRC, replies, and leases.
- [x] Cover every semantic and ownership matrix row.
- Verify: `go test ./protocol`.
- Files: `protocol/receiver.go`, `protocol/receiver_test.go`.
- Dependencies: Task 1.

## Task 3: Protocol resource gates

- [x] Add live TinyGo check and empty baseline probes.
- [x] Add fail-closed shell gate for allocations, stack, RAM, and flash.
- Verify: `./scripts/check-protocol-resources.sh`.
- Files: `cmd/protocol-check/main.go`, `cmd/protocol-empty/main.go`,
  `scripts/check-protocol-resources.sh`.
- Dependencies: Tasks 1-2.

## Task 4: Panel driver

- [x] Port the exact GDEY075T7 sequence behind the approved IO contract.
- [x] Cover commands, timing budgets, recovery, re-entry, and allocation behavior.
- Verify: `go test ./panel` and TinyGo device build.
- Files: `panel/driver.go`, `panel/errors.go`, `panel/driver_test.go`,
  `cmd/device/main.go`.
- Dependencies: none.

## Task 5: Physical panel acceptance

- [ ] Move HAT PWR from GP16 to GP15 while unpowered.
- [ ] Flash A, B, and checker artifacts with recorded hashes/toolchain.
- [ ] Confirm visible patterns and BUSY timing, with at least 180 seconds between
  full refreshes.
- Verify: physical checklist and captured results.
- Dependencies: Task 4 and user-accessible hardware.

## Checkpoint: Foundations

- [ ] Tasks 1-5 pass their verification gates.

## Task 6: Device runtime

- [x] Specify and implement frame scheduling, coalescing, leasing, and statuses.
- [x] Prove malformed/incomplete frames never invoke panel refresh.
- Verify: runtime unit and integration tests with fake transports/panel.
- Dependencies: Tasks 1, 2, and 4.

## Task 7: USB transport

- [x] Implement DTR-gated TinyGo USB CDC stream adapter without diagnostics.
- [ ] Handle partial input, backpressure, detach, timeout, and session epochs.
- Verify: host simulation plus physical USB transfer tests.
- Dependencies: Tasks 1-3 and 6.

## Task 8: Raw-frame Go client

- [x] Implement USB device discovery and exact protocol transfer.
- [x] Add TCP selection while defaulting to USB when both are available.
- Verify: client unit tests and USB end-to-end transfer.
- Dependencies: Tasks 1-2 and 7.

## Task 9: Image conversion

- [x] Decode supported host image formats and produce the exact panel frame.
- [x] Add fit/fill, rotation, threshold, and deterministic dithering.
- Verify: pixel-exact golden images and end-to-end USB display.
- Dependencies: Task 8.

## Checkpoint: USB End-to-end

- [ ] Twenty consecutive transfers and all recovery paths pass.

## Task 10: Secure Wi-Fi specification

- [x] Specify credential provisioning, authentication, replay protection,
  integrity, connection limits, timeouts, and discovery.
- Verify: threat model and automatically approved spec.
- Dependencies: Tasks 1-2 and 6.

## Task 11: Wi-Fi transport

- [x] Implement one bounded CYW43439/lneto TCP listener.
- [x] Keep Wi-Fi disabled when secure configuration is absent.
- Verify: host tests, TinyGo build/resource checks, and hardware TCP transfer.
- Dependencies: Task 10.

## Task 12: USB priority and Wi-Fi acceptance

- [ ] Prove USB preempts incomplete Wi-Fi without interrupting refresh.
- [ ] Prove authentication and replay rejection.
- [ ] Meet transfer and radio-on acceptance samples.
- Verify: physical concurrency/security/performance checklist.
- Dependencies: Tasks 7 and 11.

## Task 13: Terminal renderer

- [x] Render text, stdin, and terminal snapshots host-side.
- [x] Keep escape-sequence handling bounded and non-executing.
- Verify: golden frames and Linux/Raspberry Pi smoke tests.
- Dependencies: Tasks 8-9.

## Task 14: User documentation

- [ ] Document wiring, build, flash, USB, Wi-Fi, client, terminal, power, and
  recovery instructions with verified links and diagrams.
- Verify: commands re-run from a clean devcontainer and hardware checklist.
- Dependencies: Tasks 5, 9, 12, and 13.

## Task 15: Final acceptance

- [ ] Run full tests/builds, code review, simplification, security review,
  resource measurements, and hardware acceptance.
- [ ] Confirm every capability-map requirement has authoritative evidence.
- Dependencies: all prior tasks.
