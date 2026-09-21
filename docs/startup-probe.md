# USB startup probe

2026-09-07. Diagnostic only; not a replacement production firmware or an
e-paper acceptance test. Retained EPS1 enumerates USB; candidate EPS2 does not.
The original 0.41.1 probe stopped inside flash_uid. After the user rebuilt the
container, the unchanged probe with TinyGo 0.42.0 / Go 1.26.8 completed UID but
returned FAIL boot_session. Retained EPS1 has been restored/Hello-verified.
Current v2 probe adds a fixed failure label; it does not repair or erase data.
Follow-up: user approved recovery; the unified 0.42.0 image verified the 16 KiB
erase, rebooted, and now answers normal EPS2 Hello/Health. See the root history
for exact artifacts. Probe v2 and the old known-good EPS1 remain preserved.

## Run and interpret

Build in the existing Dev Container, from this experiment:

```sh
GOTOOLCHAIN=go1.26.8 go test -race -cover ./cmd/startup-probe ./screenboot
GOTOOLCHAIN=go1.26.8 tinygo build -target=pico2-w -scheduler=tasks -o build/startup-probe-v2-tinygo-0.42.0.uf2 ./cmd/startup-probe
```

Use only the agreed direct macOS flashing path after observing RP2350 BOOTSEL.
The preserved recovery UF2 is `build/stream/stream-device.uf2`, SHA-256
`a17ae2a2116c20f7af09d615c20ca7b006d42cb93d1a5240ff2c7eaa9a92185f`.
Do not overwrite the immutable candidate package or change wiring/HAT/FPC.

One USB client at 115200 baud with DTR asserted owns this text session:

1. Read `READY startup-probe-v2`. If there is no CDC, no action stage has been
   admitted: investigate earlier initialization/build/runtime, not panel refresh.
2. Send the single byte `n`, then **read** `BEGIN <stage>`.
3. Only after seeing that line send the single byte `r`.
4. Read `DONE <stage>` or `FAIL <stage>`. Only DONE permits the next `n`.
   If no response within the host's bounded diagnostic wait, stop, record the
   last BEGIN, and recover the retained firmware. Never automatically replay `r`.

| Stage | Action and side effects |
| --- | --- |
| `panel_io` | Same accepted pins/SPI setup; PWR low, no reset sequence or refresh |
| `flash_uid` | Actual interrupt-guarded `pico2w.UID`; no ID bytes logged |
| `boot_session` | Actual `screenboot.New`: read credentials/journal, reserve one durable epoch; no credential mutation or full erase |
| `usb_writer` | Actual bounded writer constructor; no display commands |
| `radio_constructor` | Claim radio pins/PIO/DMA; **no radio Init, Join or network worker** |

An epoch reservation may rotate/erase one epoch-journal block according to the
existing store; it is not read-only. A recovery-only boot returns FAIL and stops.
Version 2 first emits `BOOT <cause>`: `flash_geometry`, `uid`, `epoch_geometry`,
`epoch_corrupt`, `epoch_exhausted`, `epoch_io`, or `unavailable`. These fixed
labels never include underlying I/O error text or flash bytes. Corrupt means
ambiguous/invalid journal evidence, not proof of physical flash damage. No
recovery erase is admitted by this diagnostic. `epoch_io` needs a further
bounded I/O-stage measurement; it does not prove corruption.
No invented lifetime, weakened authentication or credential logging. An observed
DTR disconnect latches STOP until reboot: TinyGo retains queued RX bytes across
DTR changes, so merely disarming could admit stale n/r commands. This polled
diagnostic is for one trusted USB client, not an adversarial authentication
boundary; edges shorter than its polling period may not be observed.
Completed/failed steps never repeat within that boot.
All five DONE results still do not test full composition, networking or EPS2.

## Why acknowledgement matters

TinyGo v0.41.1 [CDC Write](https://github.com/tinygo-org/tinygo/blob/v0.41.1/src/machine/usb/cdc/usbcdc.go)
queues bytes and can return before the host receives them. An immediate fatal
flash call could hide the last log. The two-command exchange lets the host
observe BEGIN first; no arbitrary sleep is treated as proof of delivery.
Only this diagnostic command emits text, never the production binary protocol.

Primary startup references: TinyGo v0.41.1
[runtime](https://github.com/tinygo-org/tinygo/blob/v0.41.1/src/runtime/runtime_rp2.go),
[RP2 flash](https://github.com/tinygo-org/tinygo/blob/v0.41.1/src/machine/machine_rp2_flash.go),
[RP2350 ROM calls](https://github.com/tinygo-org/tinygo/blob/v0.41.1/src/machine/machine_rp2350_rom.go).
The candidate ELF places `flash_do_cmd` and its direct flash helpers in SRAM;
that static check is not proof they execute correctly on this board.

## Matching upstream issue and next controlled comparison

Verified 2026-09-07 against the GitHub API and installed toolchain source:

- [TinyGo #5408](https://github.com/tinygo-org/tinygo/issues/5408) reports
  `machine.DeviceID()` hanging a Tiny2350 board. Same MCU family and API as
  our failing boundary, but not the exact Pico 2 W board.
- [Merged PR #5413](https://github.com/tinygo-org/tinygo/pull/5413), commit
  [2747027ef261ed6c270818ffe32cb155fd3de15c](https://github.com/tinygo-org/tinygo/commit/2747027ef261ed6c270818ffe32cb155fd3de15c),
  fixes that issue and [#5412](https://github.com/tinygo-org/tinygo/issues/5412).
  It changes ROM table pointer-size handling and flash chip-select control
  to the RP2350 QMI register. Installed 0.41.1 still has both older paths.
- The [official Pico SDK flash implementation](https://github.com/raspberrypi/pico-sdk/blob/master/src/rp2_common/hardware_flash/flash.c)
  corroborates RP2350 QMI chip-select control; its BOOTRAM copy is not by itself
  anomalous. The upstream fix changes two mechanisms, so do not attribute our
  failure uniquely to chip select without another measurement.
- Stable [TinyGo 0.42.0](https://github.com/tinygo-org/tinygo/releases/tag/v0.42.0),
  published 2026-09-01, includes the RP2350 fix. This is a strong candidate
  explanation, not yet a physically verified remedy on this Pico.

Next: recover retained EPS1, persist/review the official toolchain upgrade in
Dev Container configuration, then rebuild this unchanged probe and compare the
same acknowledged stages. Do not patch the installed runtime, replace UID with
a constant, bypass durable epochs, disable security tests, or call the candidate
accepted. An environment-only install is not a completed dependency upgrade;
the user must perform any required Dev Container rebuild.

Recovery attempt after the hang: the single native 1200-baud `stty` command did
not complete and was interrupted (exit 130). No BOOTSEL volume appeared. The
serial client and reset command are closed; manual BOOTSEL reconnect is needed
before any further write. Merely retaining a `/dev/cu.usbmodem1101` node does
not establish that the firmware still services USB.

Follow-up recovery, 2026-09-07: after manual BOOTSEL, one retained-EPS1 copy
reported an extended-attribute error (exit 1), but CDC returned. A separate
single-Hello Go check validated EPS1 reply CRC/identity and idle/800x480,
exit 0. No second write or panel refresh. EPS1 is now installed; this probe
artifact remains preserved on disk. The Dockerfile pin is updated to the
official 0.42.0 image; user rebuild and repeat of the unchanged probe are pending.

## Evidence and limits

### Follow-up, TinyGo 0.42.0 / Go 1.26.8

- Installed source contains the official RP2350 fix. Unchanged v1 probe
  SHA-256 `dec336eb94e4ad797fe47e26b7c1076b1d0218c7d6b63ab02d31ccf4108c9735`
  was copied once, exit 0, followed by READY, DONE panel_io, DONE flash_uid,
  and FAIL boot_session through separate acknowledged commands. No later
  stages/refresh/join. This supports the upstream UID remedy without isolating
  a particular upstream instruction or accepting the full firmware.
- Image-bundled Go 1.27 panicked pinned staticcheck at `*ast.KeyValueExpr` in
  dependency `poll`. Exact `GOTOOLCHAIN=go1.26.8` retains the project toolchain;
  full task gate passed in 57s. Dockerfile persists this selector for the next
  user rebuild; current commands supply it explicitly. No checker was disabled.
  Official selection: https://go.dev/doc/toolchain.
- Retained EPS1 restored with native `cp -X`, exit 0; one Hello verified
  idle/800x480/CRC. Old v1 artifacts and immutable candidate remain intact.
- V2 adds test-first, fixed-cause diagnostics. Tests require corrupted storage
  to remain byte-identical, admission to remain fail-closed, and writer errors
  to propagate. Full task gate passed in 27s, changed/total coverage 91.7%.
  Artifact SHA-256:
  `5df376e92a70241646de8132a50b92f345b314f1158420c31f0f81c78bef8794`.
- V2 physical run: copy exit 0, READY v2, DONE panel_io, DONE flash_uid, then
  `BOOT epoch_corrupt` / `FAIL boot_session`. No later stages or automatic retry.
  Existing epoch code uses ErrCorrupt for scan-record or write-verify mismatch;
  exact internal subcase and data origin remain unknown. This does not diagnose
  damaged silicon. Destructive recovery is awaiting explicit user approval;
  credentials and journal must be reset together, followed by fresh enrollment.
  After this run retained EPS1 was restored again (`cp -X` exit 0), and one
  native Hello confirmed idle/800x480/CRC. V2 probe is preserved on disk.

### Historical TinyGo 0.41.1 result

- Physical run, 2026-09-07: one direct UF2 copy exited 0; CDC returned. With
  separate n/observed-BEGIN/r exchanges, the host read:

  ```text
  READY startup-probe-v1; n=prepare r=run; no display or Wi-Fi join
  BEGIN panel_io
  DONE panel_io
  BEGIN flash_uid
  ```

  After acknowledging flash_uid, no DONE/FAIL arrived during the bounded
  observation (still absent at 09:43:46 UTC). No retry was issued. This is the
  first observed nonreturning boundary, not proof of an exact instruction or
  upstream root cause. Later stages, epoch writes and refresh were not run.
- RED reproduced missing host acknowledgement; GREEN unit/race tests cover the
  command sequencer at 100%, including output failure, action failure and replay.
- Focused lint and Pico 2 W build pass; UF2 SHA-256:
  `07755cc061d0770b937c0e9750529a941f3c42bc02d10e7fb1d5badafd0f2a9e`.
- Final full task quality gate passed in 28s: changed coverage 91.7%, total
  92.2%; report `build/startup-probe-quality.log`. No tests were disabled.
- This is a reduced import/call closure, not a byte-identical candidate. A
  successful probe narrows investigation; it does not certify candidate boot.
- Hardware-only adapter execution, USB electrical behavior and actual early
  startup are not covered by host percentages. Physical results belong in the
  root [debugging history](../../../docs/epaper-debugging-history.md).
