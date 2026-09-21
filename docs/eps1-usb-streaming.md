# EPS1: isolated USB streaming experiment

Status: four successful physical full-frame transfers, including recovery
after tested first- and second-plane interruptions; not production acceptance. This does not
replace the working buffered USB firmware or freeze the manager protocol.

## Boundary

`epaperstream` reads bounded local HTML → Blitz resolves the complete 800×480
opaque monochrome target → `streamwire` sends two identical logical passes →
`streamrx` verifies ordering and SHA-256 → `panel.Stream` writes SPI.
The adapter inverts pass 0 for command `0x10`, writes pass 1 to `0x13`, and
refreshes only after verified Commit. The new Pico path has no full framebuffer.

Modules: host renderer/client, wire codec, transaction receiver, USB owner,
panel adapter, TinyGo pin binding. Only the adapter knows controller commands.
Supported here: two-pass full frame. Not implemented: partial, mask/copy/cache,
arbitrary regions, timestamp corner, authenticated manager transport or Wi-Fi.

## Wire contract

Integers are little-endian. Header: 32 bytes. Maximum payload: 100 bytes.

| Offset | Field |
| --- | --- |
| 0–3 | ASCII `EPS1` |
| 4 | Hello=1, Begin=2, Data=3, Commit=4, Query=5, Abort=6, Reply=7 |
| 5 | Pass index |
| 6–7 | Payload length |
| 8–15 | Nonzero session epoch |
| 16–23 | Intent ID |
| 24–27 | Logical byte offset within pass |
| 28–31 | CRC32 IEEE over header bytes 0–27 then payload |

Hello: zero ID/pass/offset, empty payload, fresh random host epoch. Begin: SHA-256
of the immutable logical frame as 32-byte payload. Data: exactly the next chunk.
Both passes must match the same digest. Commit/Query/Abort: empty payload and
zero pass/offset. Reply echoes request epoch/ID.

Reply payload (20 bytes): result at 0, state at 1, next pass at 2, reserved at 3,
next offset uint32 at 4, width uint16 at 8, height uint16 at 10, chunk maximum
uint16 at 12, passes at 14, reserved at 15, minimum full interval seconds uint32
at 16. Results: 0 success, 1 framing, 2 state/identity, 3 chunk, 4 digest,
5 timeout, 6 deferred, 7 sink/other. Code 7 does not yet carry panel phase/command.
States: Idle=0, Receiving=1, Ready=2, Complete=3, Failed=4, Closed=5.

CRC/SHA provide consistency, **not authentication**. Trusted physical USB only;
do not expose EPS1 over TCP, Wi-Fi or the public internet.

One request waits for one reply. With 100-byte chunks, two 800×480 passes carry
96,000 pixel bytes, 126,848 request bytes and 50,076 reply bytes: 176,924 total
application bytes, excluding USB overhead. These are format calculations, not
measured throughput. This is not the final bandwidth-optimized protocol.

## Failure and power ownership

- No automatic retry after ambiguous completion. Latest completed intent can
  replay status within its receiver; reboot does not preserve that outcome.
- Corrupt framing closes the link; semantic rejection invalidates staging.
  A subsequent Commit cannot authorize the rejected image.
- DTR drop aborts and drains queued RX. Failed links discard input until DTR
  reset; no scanning arbitrary payload for the next magic header.
- Input budgets: 20 seconds idle, 120 seconds total. SPI/BUSY keeps the existing
  independently bounded lifecycle. Timeout does not diagnose a wiring fault.
- Installed TinyGo 0.41.1 CDC Write waits for TX space, yielding through
  `runtime.Gosched`. One worker with one copied reply isolates the wait. After
  one second the owner returns an error and aborts panel power. The writer
  remains unusable until MCU reboot: it cannot cancel the driver call or reuse
  that buffer safely. At most one worker remains blocked. DTR is not recovery
  for this specific failure. This guarantee still requires a target test.
- Experimental full-refresh minimum: 180 seconds, retained across DTR reconnect
  but not power loss. Boot waits for the server; no boot picture or clear.

## Evidence and artifacts — 2026-09-05

- Quality task PASS, 22 seconds; changed executable-line coverage 91.0%, total
  89.3% (required baseline 75%). No gate exclusions or weakened thresholds.
- Focused race tests pass. SPI oracle compares every buffered/streamed event;
  fault injection fails every SPI write independently. Tests cover wrong data,
  identity, replies, expired operations, stale RX and an active Begin with a
  blocked USB reply causing Abort. Independent review: no remaining required
  code findings; reviewer suggested the now-added blocked-Begin regression.
- TinyGo 0.41.1 `pico2-w` build: flash 96,152 bytes, static RAM 7,964 bytes.
  This is not peak heap/stack or evidence of improved energy/performance.
- Ignored artifacts: `build/stream/stream-device.uf2`,
  `build/stream/epaperstream-linux-arm64`, `build/stream/epaperstream-macos-arm64`.
  macOS client is cross-built, not runtime-tested; it also needs a compatible
  Blitz worker. The existing Linux worker cannot run as a macOS executable.

Client arguments: `epaperstream WORKER FILE.html SERIAL_PORT`. HTML limit 32 KiB;
serial reads 60 seconds, whole transfer 180 seconds. Use a fresh USB session
without another serial reader. `stream=committed` is not visible acceptance.
All project builds/tests ran in the existing Dev Container. At that software
checkpoint no Pico was flashed; the subsequent device test is recorded below.

Remaining: repeated physical updates and stopped-reader/reconnect tests, target
peak RAM and powered-duration/energy measurement, timestamp, authenticated
manager integration, final protocol, partial-refresh acceptance and reconnect
outcome reconciliation. Preserve the known-good artifact before a target test.

Sources inspected: [pinned Waveshare V2 reference](https://github.com/waveshareteam/Pico_ePaper_Code/blob/c9bcd84db5adf5f085353649a8a5c31492bc5fb8/c/lib/e-Paper/EPD_7in5_V2.c),
the known-good project driver and installed TinyGo
`src/machine/usb/cdc/usbcdc.go` (Write and Gosched link). Upstream reference:
[TinyGo CDC](https://github.com/tinygo-org/tinygo/blob/v0.41.1/src/machine/usb/cdc/usbcdc.go).
This TinyGo installation has no Git metadata; the USB finding is from direct
installed-source inspection, not a verified local source commit hash.

## First device transfer — 2026-09-05

User authorized direct macOS USB access and flashing. Existing recovery UF2
`dist/remote-epaper-usb.uf2` still matches known-good SHA-256
`8d9430b8feb76a3811e6308e9b14c1ca1cfa325dbe906348318cd64c1fa676aa`.
CDC 1200 baud entered RP2350 bootloader; INFO_UF2 identified Raspberry Pi RP2350.
Copied streaming UF2 SHA-256
`a17ae2a2116c20f7af09d615c20ca7b006d42cb93d1a5240ff2c7eaa9a92185f`
directly to `/Volumes/RP2350`. CDC returned as `/dev/cu.usbmodem2101`.

The cross-built macOS client used ignored `build/stream/blitz-container` to
invoke the existing Linux Blitz worker through `docker exec -i`, without USB
passthrough. One `build/stream/acceptance.html` frame, headed `STREAM USB` with
badge `ТЕСТ S1 · 05.09.2026`, returned `stream=committed`, exit 0.
At transfer completion, visible output was still awaiting confirmation.
The user then answered «так» when asked whether `STREAM USB` and the S1 badge
were visible: **the first complete HTML → Blitz → USB → streamed SPI → visible
panel image is physically confirmed on Pico 2 W / 7.5-inch V2 / HAT Rev2.3.**
This accepts one full-frame transfer, not repeated-update reliability or fault
recovery, and does not measure energy or peak RAM.
No automatic retry, second refresh, wiring change or fault injection performed.
The badge is a fixed test identifier, not implemented last-full timestamp state.

## Second transfer — 2026-09-05

User requested a repeat update. Sent `build/stream/acceptance-2.html`, headed
`STREAM USB — 2` with badge `ТЕСТ S2 · 05.09.2026`, through the same direct
macOS client and container worker. No flashing, USB unplug, reset or firmware
change was performed by the agent between S1 and S2. The client opened a new
CDC session; the firmware's refresh interval was not changed or bypassed.
Result: exit 0, `stream=committed`. Visible acceptance initially awaited the
user; the user then confirmed «так» to seeing `STREAM USB — 2` and `ТЕСТ S2`.
Two distinct complete frames are now physically confirmed, including a repeat
update through a new CDC session without reflashing. This is not a long-run
reliability, interrupted-transfer recovery, partial-refresh or energy test.

## Controlled interruption — 2026-09-05

Added host-only `--interrupt-after-first-chunk` before the existing arguments.
It uses the normal client validation, permits the first Data record, consumes
and validates its reply, then refuses the next Data record. Commit is explicitly
blocked. The normal port cleanup drops DTR and closes the session. Real protocol
errors are not reported as a successful deliberate interruption.

Tests prove no Commit reaches the receiver, the reply is consumed, a subsequent
full upload works in the host fixture, and the CLI drops DTR/closes the port.
Focused race tests pass; quality task PASS (17 seconds). Firmware unchanged.

On the physical Pico, attempted S1 content while S2 was visible. Result: exit 0,
`stream=interrupted after first acknowledged chunk; no commit sent; closing DTR`.
For the current negotiated 100-byte chunk, only 100 of 48,000 first-plane bytes
were accepted. The user then confirmed «так»: S2 remained unchanged after the
interruption. This physically confirms preservation of the visible frame for
this early first-plane interruption, not every possible interruption point.
Actual PWR voltage was not measured. At this checkpoint recovery was untested;
the subsequent recovery upload and observation are recorded below.

## Recovery upload after interruption — 2026-09-05

Next, user authorized the full S3 upload. Sent `build/stream/acceptance-3.html`
with heading `STREAM USB — 3` and badge `ТЕСТ S3 · 05.09.2026`, using the same
direct macOS USB client and container worker. No agent-triggered reset,
reflashing, unplugging or limiter change between interruption and recovery.
Client exit 0: `stream=committed`. The user then confirmed «так» to seeing
`STREAM USB — 3` and `ТЕСТ S3`: visible recovery is confirmed for this early
first-plane interruption followed by a new complete upload without reboot.
No further update was sent. Three distinct frames (S1, S2, S3) are physically
confirmed. Other interruption points, lost completion replies, power-loss
recovery, long-run reliability and energy measurements remain separate gates.

## Second-plane interruption — 2026-09-05

Extended the host-only probe with `--interrupt-in-second-plane`. It allows the
complete first plane, then one acknowledged second-plane fragment. Every
subsequent record is rejected locally, including Commit. The early interruption
flag still works. Host tests check exact accepted bytes in both planes, zero
commits, consumed acknowledgements and fresh-upload recovery. Race tests pass;
quality task PASS (16 seconds), changed coverage 91.2%, total 89.3%.

Physical attempt: S1 content while confirmed S3 remains displayed. Client exit 0:
`stream=interrupted pass=1 after first acknowledged chunk; no commit sent; closing DTR`.
With the current 100-byte chunks, this means 48,000 logical first-plane bytes
and 100 second-plane bytes acknowledged. Firmware, wiring and refresh limiter
unchanged; no reset or reflash. The user confirmed «так» to seeing unchanged
`STREAM USB — 3` / `ТЕСТ S3`. Visible preservation is confirmed for this
second-plane interruption, in addition to the earlier first-plane case.
At this checkpoint second-plane recovery was unverified; its subsequent test
is recorded below. Measured PWR shutdown remains unverified on hardware.

## Recovery after second-plane interruption — 2026-09-05

Sent `build/stream/acceptance-4.html` (`STREAM USB — 4`, `ТЕСТ S4`) through the
same direct macOS client/container renderer after the second-plane cut.
No agent-triggered reboot, reflash, unplug, wiring or limiter change.
Client exit 0, `stream=committed`; protocol-level recovery succeeded.
The user confirmed «є» to seeing `STREAM USB — 4` / `ТЕСТ S4`.
Visible recovery after this second-plane interruption is therefore confirmed,
without an agent-triggered restart. Four distinct frames are physically
confirmed, with image preservation and recovery at both tested interruption
points. This does not cover all interruption points, lost completion replies,
power loss, long-run reliability or energy/peak-RAM measurements.
No further refresh sent.

## Manager integration and serial timeout correction — 2026-09-05

An opt-in loopback HTTPS manager composes batching, Blitz and the same client
through `epaperstream --raw`. Raw mode reads one bounded local BZM1 frame; it
does not interpret HTML or spawn a worker. See `screen-manager-usb.md` for the
API and host-test evidence. No new firmware or physical update; S4 remains.

Earlier close-timer wording is not proof that every blocked OS write ends.
Pinned serial v1.8.0 explicitly wakes pending Unix reads on Close; Write uses
a blocking syscall. Manager supervision additionally cancels/reaps the direct
child using CommandContext and WaitDelay. Tests establish process cancellation,
not every cable/kernel fault. Standalone epaperstream retains its best-effort
close timer; the manager never automatically retries an ambiguous result.

## S5 through the new native manager — 2026-09-06

With explicit user permission, sent one scene headed `MANAGER S5` through the
new localhost HTTPS API, actual Blitz container worker and supervised direct
macOS USB client. No firmware or wiring change, no retry, existing 180s guard.
After startup cooldown, API status reached
`{"confirmed":1,"current":1,"in_flight":0}` by 00:43:25 Europe/Kyiv.
Manager was then stopped cleanly. The user subsequently confirmed «так» to
seeing `MANAGER S5` and `ТЕСТ S5 · 06.09.2026` on the physical panel, completing
visible acceptance separately from terminal protocol success. S5 is now the
latest known-good manager-to-display checkpoint; Wi-Fi, partial refresh and
remaining fault/resource gates stay open. See `screen-manager-usb.md` for the
preceding client/certificate compatibility failure and its verified resolution.
