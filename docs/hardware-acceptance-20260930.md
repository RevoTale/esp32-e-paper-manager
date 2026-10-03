# ESP32 7.5 V2 acceptance — 2026-09-30

This checkpoint verifies one USB and one encrypted Wi-Fi frame from the
standalone checkout. It does not close production acceptance or partial refresh.
The user confirmed both images and their updated corner timestamps.

## Artifact and write boundary

- Source revision: `1b60448ddf2fd594bf3e05d3ceafc593f28412c3`, clean at build/use.
- Quality run [36629806772](https://github.com/RevoTale/esp32-e-paper-manager/actions/runs/36629806772)
  passed. CI and local firmware are not claimed byte-identical.
- Local `firmware/esp32/build-release/epaper_receiver.bin`: 772400 bytes;
  SHA-256 `0002317a05b5170b9928e72b5ad6aa7f8f6b255c329301aeac0f72f2d52211ef`.
- Existing native espflasher wrote only application offset `0x10000`.
  No erase-all, bootloader, partition-table or credential-sector write.
- Flasher reported an image digest update and successful read-back MD5
  `bbb53139fc811c10032782e39284b6da`. This is the written-image evidence;
  it must not be confused with the input-file SHA-256.
- Three compressed blocks timed out on their first attempts. Built-in retries
  completed and final verification passed. Cause was not established.

## USB result and unresolved startup symptom

New and retained clients initially returned EOF. The port existed and no
competing serial process was found. A subsequent status query succeeded with
800×480, two passes, chunk1000 and a remaining startup guard of103718ms.
No second flash, wiring change or physical power cycle was performed.

The new macOS client rendered `firmware/esp32/usb-acceptance.html`, waited for
readiness and returned `screen=confirmed`. The user confirmed `ESP32 USB READY`
and the new timestamp. The template's literal2026-09-20 is historical text,
not this test's execution date. Initial EOF and longer-than-initially-estimated
wait remain unexplained; successful recovery is not a root-cause fix.

## Wi-Fi configuration and result

Read-only USB Inspect proved the stored manager address no longer matched the
Mac's DHCP address. With explicit permission, `rotate` changed the endpoint
and device key, retained device identity and Wi-Fi credentials, and returned
`code=0 state=1 generation=2`. A new private enrollment file was promoted only
after the matching acknowledgment. The old enrollment is no longer active.
No keys, credentials or private network addresses belong in tracked evidence.

The matching Go manager ran natively on macOS with HTTPS bound to loopback,
the existing TLS/API credentials, new enrollment and Wi-Fi-only screen transport.
Normal/urgent policies remained180s/30s. No USB image was sent during this test.
One ETag-guarded PUT submitted `firmware/esp32/wifi-acceptance.html` as revision1.

Terminal API evidence:

- `current=confirmed=delivered=1`, `in_flight=0`, `refresh_trusted=true`.
- Full cycle1: `2026-09-29T22:05:58.632247Z` through
  `2026-09-29T22:06:04.375077Z`:5.742830s. These UTC times fall on September30
  in Europe/Kyiv. They measure the reported refresh cycle, not end-to-end latency.
- User confirmed `ESP32 WIFI READY / TEST FRAME B` and new corner time.

Manager was left running after acceptance; that is a checkpoint, not a promise
that it remains alive. API state/scenes are volatile. Recheck process and status
before further work. A future DHCP address change will again require endpoint
management; silently rotating keys is not an address-update strategy.

## Remaining work

1. Qualify panel-specific partial refresh; current firmware remains full-only.
2. Diagnose post-flasher EOF with bounded, secret-safe transport evidence.
3. Verify cold start, reconnect, interrupted staging, ambiguous completion and
   recovery on this candidate without unsafe automatic physical refresh replay.
4. Measure device RAM/stack headroom, timing and power; host gates are not proof.
5. Close packaging/dependency-notice issues and test the packaged manager with
   real hardware before publishing or declaring production acceptance.

References: [receiver contract](../firmware/esp32/README.md),
[refresh contract](refresh-priority.md), [release notices](runtime-dependency-notices.md).

## Subsequent host-only cancellation correction

The later partial-window quality run failed in
`TestServeClosesListenerAndBoundedHandshakeOnCancellation`: `Serve` returned
`net.ErrClosed` instead of `context.Canceled`. No hardware test caused this.

Cause: cancellation's callback can close the hub after `Accept` returns a
socket but before `reserve` admits it. The Accept-error branch checked the
context; the reserve-error branch did not. Both still closed resources.
The fix preserves context cancellation in both branches without masking an
independent hub close when the context is live.

Regression: `TestServeClosureDuringAdmissionPreservesCause` forces each order
without sleeps, checks the precise error and proves the unregistered socket
closes. Its canceled case failed before the fix. Afterward, the entire
`screenhub` package passed100 runs with the race detector. This is not a
diagnosis of the earlier USB EOF, which remains unresolved.

The subsequent complete `make quality` exited0, including three independent
coverage runs. Evidence: local `build/quality-partial-and-cancellation.log`.
No flash, service restart, new display frame, commit or push was performed.
