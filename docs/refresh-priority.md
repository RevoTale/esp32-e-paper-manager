# Refresh priority contract

Approved direction, 2026-09-21; implemented software candidate, not yet
physically accepted or deployed as a release.

HTTP metadata on PUT/PATCH, not HTML properties:

- `X-Update-Priority: normal | urgent` (default normal).
- `X-Refresh-Mode: auto | partial | full` (default auto).
- Reject unknown, empty explicit, duplicate or comma-separated values before
  changing the canonical scene. Authentication and If-Match still apply.
- Metadata belongs to the same immutable scene revision as its pixels. Latest
  replacement wins, including priority; do not retain urgency from a replaced
  document. Multi-edit PATCH remains atomic. Maintenance uses normal priority.
- Urgent bypasses optional batching and normal cadence, not an active writer,
  BUSY, invalid staging, digest validation, a failed panel, or reconciliation
  after an unknown result. It must have a separate configurable rate budget.
- Full/partial is independent of urgency. Auto chooses a supported mode;
  explicit unsupported partial fails, never silently becomes full. The current
  ESP32 7.5 V2 adapter supports full only.
- 180 seconds remains the conservative normal default, not a claim of an
  absolute physical limit. A shorter policy is an explicit operator override,
  not a manufacturer-backed safety guarantee. Source: Waveshare
  https://www.waveshare.com/wiki/7.5inch_e-Paper_HAT_Manual
- Freshly connected legacy firmware cannot silently ignore urgent metadata.
  Negotiate support before sending the new transaction form. Preserve the
  current Pico wire behavior and credential storage.

Implementation order / acceptance gates:

1. Typed metadata and pure scheduling tests; malformed inputs and no side effects.
2. Atomic PUT/PATCH revision metadata, pending/in-flight ownership, batching.
3. Negotiated screen transaction options; Go/C interoperability and old-peer rejection.
4. Manager and firmware cadence policy, busy/replay/disconnect guards.
5. CLI configuration, status, documentation, full quality/build checks.
6. Physical USB then Wi-Fi acceptance; do not conflate ACK with visible output.

This explicitly supersedes historical wording that all refresh cadence is an
unconditional firmware prohibition. Electrical/controller safety and quality
constraints remain unchanged. No partial waveform is enabled by this decision.

## Operator configuration and behavior

Start the new manager with `-refresh-policy`. This explicitly opts into an
operator cadence override requiring the new ESP32 receiver. Existing commands
without this flag retain the legacy behavior, including Pico compatibility.

- `-full-interval 180s`: normal budget.
- `-urgent-interval 30s`: separate urgent budget; a project default, not a
  manufacturer-qualified interval. Both flags are configurable, whole
  milliseconds, positive, at most `4294967295ms` (wire integer bound).
- `-debounce-interval` and `-max-wait` stay optional; urgent skips both.
- Startup and unknown completion retain a conservative guard of at least the
  advertised legacy interval and the configured interval. After a known
  successful completion, urgent uses its own budget measured from completion.
- An unchanged bitmap remains a no-op. `full` selects the physical mode; it
  does not mean refresh identical pixels or retry an unknown transaction.
- Periodic maintenance and recovery are normal full cycles. The timestamp
  remains the full-cycle timestamp, not the time a request was accepted.

For example, add `X-Update-Priority: urgent` and `X-Refresh-Mode: full` to an
otherwise unchanged authenticated conditional PUT/PATCH. Do not put urgency
inside HTML. `202` means accepted scene revision, not visible panel completion.
Bad metadata returns `400 invalid_refresh_options`; partial or urgent without
the negotiated-policy manager returns `501 refresh_unsupported` before mutation.
Normal replacements clear an earlier pending urgency; in-flight metadata cannot
be changed by a later request.

Short intervals are permitted intentionally, not certified safe. The encoding
minimum is not a physical panel rating. No throughput, lifetime, energy saving,
or partial-refresh capability is inferred from passing host tests.

## EPS2 extension and upgrade order

`FeatureRefreshPolicy = 0x0008` in Hello explicitly advertises `BeginRefresh`
(kind 13). Its 44-byte payload contains the existing 32-byte frame digest,
priority at byte32 (`0 normal`, `1 urgent`), mode at33 (`0 auto`, `1 partial`,
`2 full`), zero reserved bytes34–35, normal interval at36–39 and urgent interval
at40–43 (little-endian uint32 milliseconds). The MCU validates everything
before staging. This full-only adapter rejects partial. Commit/Query retain
the original digest and transaction ID; consumed IDs never trigger a new frame.
The extension adds 12 bytes per transaction, no additional pixel storage.

**Upgrade manager and USB host tools together with the firmware.** Old Go
clients reject unknown capability bits; they will fail closed against this
new ESP32 image. New hosts still support old firmware without `-refresh-policy`.
With that flag they reject an old peer before pixel staging, never silently
downgrading urgent. The new feature is not advertised by the unchanged Pico
receiver. Credentials, EPN2 authentication and encryption remain unchanged.

## Evidence

- Typed parsing, replacement/lease metadata, batching, cadence, CLI opt-in,
  old-peer rejection and client error paths have automated regression tests.
- Native C tests verify independent urgent budget, unsupported partial,
  active transaction, consumed ID and fatal-panel guards.
- Go/C interoperability sends two complete frames using both legacy Begin and
  BeginRefresh into the real receiver/panel code with simulated physical I/O.
- These checks do not prove visible output or resolve the earlier unconfirmed
  USB-v3 candidate Wi-Fi acceptance. Hardware verification remains separate.

## Candidate artifact and acceptance checkpoint

Built in the existing Dev Container, without replacing the retained checkpoint:

- ESP32 application: `experiments/12-esp32-epaper/receiver/build-refresh-policy/epaper_receiver.bin`.
- SHA256: `d96b45e427116f383d15faea7c0140b55b6caac2601fd099e262fedf49f73dbd`.
- Size: 772400 bytes. Application-only offset: `0x10000`.
- macOS arm64 tools under `experiments/12-esp32-epaper/.local-mac/`:
  `epaper-manager-refresh`, `epaperscreen-refresh`, `epaperprovision-refresh`.
- Candidate generated SDK header and partition-table binary match build-release
  byte-for-byte. The retained build-release SHA256 is still
  `084cc044e996ee70154665446c99ad6198bbe79f314e9158937a1cb31e5c85f4`.

Reproduce the firmware build inside the existing Dev Container with ESP-IDF
environment loaded, from `experiments/12-esp32-epaper/receiver`:

```sh
idf.py -B build-refresh-policy -D SDKCONFIG=build-refresh-policy/sdkconfig build
```

Use the isolated SDK config, not the ignored root config. CMake rejects panic
printing and core dumps. Do not use a blanket erase/full-flash command for an
already provisioned board. Preserve the last16KiB credential/epoch sectors.

Remaining physical acceptance, in order:

1. Confirm ESP32/panel is connected and idle; stop its manager before serial.
2. Flash only the verified application with the existing native macOS flasher;
   compare read-back MD5 (`708b61f49df59df1f97933dca08c2aff`).
3. Updated USB tool: read-only capabilities/status, then a distinct visible
   frame through the updated manager with policy enabled. Respect the initial
   unknown-completion guard; do not conflate ACK with visible acceptance.
   Use `--status esp32:/dev/cu.YOUR_ESP32` and manager
   `-serial esp32:/dev/cu.YOUR_ESP32`; a bare path selects the Pico serial
   lifecycle and is not a valid ESP32 acceptance probe.
4. Submit normal and urgent scene changes, verify the urgent budget and that a
   later normal replacement does not inherit urgency. No refresh during BUSY.
5. Close serial, run encrypted Wi-Fi acceptance with the existing enrollment;
   if it stalls, diagnose the recorded radio-join/handshake boundary separately.
6. Record visible user observation and interruption/reconciliation results
   before calling this a hardware-accepted release.
