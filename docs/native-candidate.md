# Native engine candidate: operator handoff

One Go manager, one generic TinyGo Pico 2 W firmware. HTML stays on the host:
HTML5 → typed inline CSS → viewport layout → Canvas CPU paint → 1-bit frame →
EPS2 Raw/PackBits → bounded Pico buffer → both controller RAM planes → refresh.
No Blitz worker, Rust runtime, GPU, Bluetooth session or on-Pico HTML parser.
See [engine profile](../SPEC-engine.md) for supported syntax and explicit limits.

## Accepted USB checkpoint

2026-09-07: the user confirmed the native dashboard after `screen=confirmed`.
Use this exact pair, relative to the experiment directory:

- First visibly accepted UF2: `build/screen-device-tinygo-0.42.0-recovery.uf2`, SHA-256
  `8b844bb0f369989e6f6639dc04f8320cdf1b2853437e2520d6485c9622b69b4a`.
  Despite its historical filename, this is the unified firmware.
- USB sender: `build/epaperscreen-usb-rx-480-darwin-arm64`, SHA-256
  `c8072286c5d6ceb2a0e5477e5fb1fc14499468d7dc79547454f0e3c80981fa74`.

The frozen `build/native-candidate.gEnGoa` package predates the TinyGo startup
fix and bounded USB sender. Keep it as evidence; do not flash its UF2 or use
its old sender for this checkpoint. Rebuilt artifacts require their own
checksum/software verification before claiming equivalence or target acceptance.
USB image acceptance does not establish WPA3, reconnect, partial or power results.

### Current hardware status and separate Linux direction (2026-09-08)

The replacement Pico has the first-accepted UF2 listed above. Its service area
was explicitly erased with user approval; after reboot EPS2 responded and one
image send completed, but the user still saw the old dashboard. Repeat optical
acceptance remains failed. The diagnostic candidate below is historical, not
currently installed. No claim that replacing Pico fixed the fault.

Preserve this Pico/TinyGo implementation and its known-good artifacts. The user
selected an additional Go application/backend for direct GPIO/SPI control on
their Raspberry Pi 5 in a Pironman homelab setup, not replacement Pico firmware.
Reuse host rendering and display abstractions where appropriate; keep Linux
GPIO ownership and deployment separate from the Pico transport and HAL.
Before implementation or connection, establish exact Pironman model, occupied
GPIO/SPI lines, OS/kernel and suitable Pi5 driver access from official sources.
Do not change cooling, storage or other homelab services or assume the40-pin
header is free. This is an approved direction, not an implemented Linux port.

### Historical diagnostic candidate

After the second image returned protocol success but remained visibly unchanged,
installed `build/screen-device-panel-trace.uf2` (SHA256
`65ce3ced1e05b854a33ece3077791ec4f57b6707aabe49fdb91f2623e18fc66a`).
At that stage the accepted checkpoint was preserved but not installed.
Use `build/epaperscreen-panel-trace-darwin-arm64 --panel-status SERIAL_PORT`
for one cached cycle report (sender SHA256
`98754795c1e6d4f31fae869bedcf948125db5fc99910a808349cc84ab3af58d9`).
No acquisition, upload, retry, or panel operation is triggered by that request;
opening USB still has normal priority effects. The
[wire contract](eps2-wire.md#additive-panel-trace-2026-09-07) explains state and
counter limits. This is observability, not a demonstrated physical repair.

## Build and package

Packaging gap: `scripts/native-release.sh` still requires historical TinyGo 0.41.1;
it currently rejects the accepted 0.42.0 environment. The command below documents
the E4 package workflow, not a verified current rebuild. Update its version and
associated notices/source checks together before producing the next package;
do not downgrade the working environment to satisfy this stale guard.

After that correction, run **inside the existing Dev Container**, from this experiment:

```sh
sh scripts/native-release.sh
```

This runs full quality gates, builds Linux/arm64 and macOS/arm64 host tools,
audits both host targets using pinned govulncheck, builds the UF2/ELF, generates
previews, collects notices/source inputs and SHA256 checksums. It prints a new
`build/native-candidate.XXXXXX` directory and never overwrites the known-good
hardware artifacts. No flashing, enrollment, listener startup or deployment.
The historical package used TinyGo0.41.1. The user subsequently rebuilt the
container with TinyGo0.42.0; the accepted UF2 above was built with Go1.26.8.
Use explicit `GOTOOLCHAIN=go1.26.8` for project commands until a later container
recreation verifies the persisted environment pin. See the current
[acceptance record](native-candidate-acceptance.md), not the frozen package copy.

The package contains `screen-device-pico2w.uf2`, `epaperscreen-*`,
`epaperprovision-*`, `epaper-manager-*`, `engine-preview-*`, `epaperctl-*`,
`dashboard.html`, previews and evidence. Check `SHA256SUMS` before using it.
The stamped preview uses a **synthetic** fixed time; USB submission samples the
actual host clock after readiness. A PNG or successful build is not panel proof.

## Continue acceptance: USB only

Keep the accepted wiring, HAT switches and power sequence unchanged. Use this
candidate UF2 only after explicitly starting hardware acceptance. BOOTSEL is
the bootloader button, not a reset button. The firmware targets **Pico 2 W**.
It waits for a sender: no automatic boot picture or unsolicited refresh.

On macOS, from this experiment with the accepted sender and an observed port:

```sh
./build/native-candidate.gEnGoa/epaperctl-darwin-arm64 -list
./build/epaperscreen-usb-rx-480-darwin-arm64 --status /dev/cu.usbmodemEXAMPLE
./build/epaperscreen-usb-rx-480-darwin-arm64 -timezone Europe/Kiev build/native-candidate.gEnGoa/dashboard.html /dev/cu.usbmodemEXAMPLE
```

Replace the example port. The last command renders and submits once, then
reports confirmed protocol completion or a concrete error. Confirm actual
pixels separately. USB has priority; opening it can abort incomplete Wi-Fi
staging. Device startup and full updates enforce the accepted **180-second**
floor, including a conservative cold-client wait; this is not a render timeout.
Do not shorten it or enable partial refresh as part of this migration.

Make a new copy of `dashboard.html`, change text there and preserve the accepted
input. Send again after readiness, then compare both
content and the bottom-right full-cycle time. A lost ACK is reconciled by exact
transaction identity, not an automatic second refresh. An unconfirmed error
requires inspecting state before another deliberate attempt. The old picture
remaining visible proves neither connection nor current firmware execution.

## Manager over USB

Use an operator-owned valid TLS certificate/key and a private mode-0600 API
token file of at least 32 bytes. Do not put secrets in command arguments.
Build the manager from current source inside the Dev Container; do not use the
frozen package manager: the USB chunk bound lives in its parent sender as well
as in the standalone CLI. A new proxy alone cannot repair an old parent client.

```sh
./epaper-manager-darwin-arm64 -screen -trusted-html \
  -screen-transport usb -usb-worker ./build/epaperscreen-usb-rx-480-darwin-arm64 \
  -serial /dev/cu.usbmodemEXAMPLE -timezone Europe/Kiev \
  -tls-cert server.crt -tls-key server.key -token-file api-token.txt
```

The author API listens on `127.0.0.1:8443`. Authenticated `GET /v2/screen`
returns the current strong ETag; `PUT /v2/screen` accepts bounded UTF-8 HTML with
that exact `If-Match`, bearer authorization and `Content-Type: text/html`.
`PATCH` provides atomic by-ID edits; `GET /v2/screen/status` separates current,
rendered and delivery evidence. See [author API](engine-authoring-api.md).
`202` means accepted for processing, not refreshed. Tokens stay in HTTPS.

Identical final pixels suppress delivery unless maintenance is due. Full
maintenance defaults to ten minutes (`-maintenance-interval 10m`), independent
of author changes but still subject to the device floor. `-debounce-interval`
and `-max-wait` are optional; zero leaves each disabled. Related edits are one
atomic scene. The manager stores current scenes in RAM; after a manager restart,
resubmit the authoritative HTML rather than trusting the retained panel image.

## Enroll WPA3 and switch manager delivery

Use [USB provisioning](provisioning.md) with an explicit `-port` on macOS.
The generic UF2 contains no credentials. Provisioning consumes private JSON on
stdin, generates a fresh device key and writes a private enrollment transaction.
Passphrase/key never belong in logs, Git or the candidate package. Keep pending
enrollment evidence after an ambiguous result; never blindly repeat rotation.

Provide a reachable `tcp://manager.example:9757` endpoint and IANA timezone.
Then start the same manager with `-screen-transport wifi -enrollment FILE`
and an explicit appropriate `-device-listen ADDRESS:9757`; omit USB-worker and
serial options. The Pico initiates the connection; no Pico inbound HTTP port
or custom public-router opening is required. The EPN2 link mutually authenticates
and encrypts EPS2 records. WPA3-SAE with required protected management frames is
the only Wi-Fi join policy; there is no WPA2 fallback.

The Pico network stack currently uses IPv4; IPv6 on Pico was explicitly deferred.
The host supports its OS networking. This is not a local-address-only protocol:
a reachable public manager endpoint can carry the same authenticated link.
Public deployment is a separate operator action, never performed by this build.
Direct non-loopback **author API** binding remains deliberately blocked pending
explicit approval; existing operator-managed HTTPS reverse-proxy deployment is
a separate configuration option, not something this task silently installs.

## Evidence boundaries / remaining manual checks

- Software tests cover complete HTML→encrypted/USB records→physical-driver
  traces, both planes, tampering, truncation, stale sessions, USB preemption,
  lost ACKs, provisioning failures and bounded resource admission.
- The first native USB image is visually accepted. Repeat update, Wi-Fi, cable
  reconnect, credential rotation/power-cut, long-running heap/stack and power
  measurement remain separate physical checks; see the live acceptance record.
- Compiler RAM summaries are not peak RAM. Read the linked-ELF/static/task/heap
  breakdown in [firmware resources](native-firmware-resources.md).
- Upstream CYW initialization/join has cancellation gaps; health exposes
  failures and USB remains the diagnostic path. See
  [Wi-Fi constraints](wifi-runtime-constraints.md). No claim of proven endless
  radio recovery, complete Bluetooth byte removal, or measured battery life.
- Partial refresh is not negotiated. Accepted full refresh, complete-before-
  visible checks and HAT power cleanup remain the baseline.

Record each physical result and exact artifact checksum in the debugging
history. Do not replace the known-good checkpoint until visible acceptance.
