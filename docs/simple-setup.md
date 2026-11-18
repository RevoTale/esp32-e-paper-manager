# Simple Docker setup

Prepare manager credentials once, then start the manager normally. No manual
certificate generation, token generation, UID changes or chmod on generated
files. ESP32 firmware and USB provisioning are separate steps.

Requires Docker Compose, an existing private USB enrollment file, and a manager
image containing `epaper-manager setup`. **v0.1.1 does not contain this command.**
The setup command is available in the [v0.2.0 release](https://github.com/RevoTale/esp32-e-paper-manager/releases/tag/v0.2.0).

## Install

Copy `deploy/compose.setup.yaml` to your deployment directory as `compose.yaml`.
Put these three non-secret values in `.env` next to it:

```dotenv
EPAPER_IMAGE=ghcr.io/revotale/esp32-e-paper-manager@sha256:REPLACE_WITH_RELEASE_DIGEST
EPAPER_ENROLLMENT_FILE=/absolute/path/to/private/device.json
EPAPER_DEVICE_BIND=192.168.1.10
```

Use a LAN IPv4 address on the Docker host that the ESP32 can reach at its
configured port. A configured DNS name must resolve to that host. Changing
Compose does not change the endpoint on the device. To move an enrolled board
from a development Mac to another host, follow the [USB endpoint update](../firmware/esp32/README.md#move-an-enrolled-receiver-to-another-manager)
with compatible firmware and CLI. The enrollment source
must be an existing regular mode-0600 file, as produced by USB provisioning;
setup does not relax or change permissions on the source.

```sh
docker compose config --quiet
docker compose --profile setup run --rm setup
docker compose up -d epaper-manager
```

Setup is offline and does not flash, connect to or reprovision ESP32. It imports
the enrollment unchanged, generates a random 256-bit API token and a one-year
self-signed Ed25519 loopback TLS certificate. It prepares private ownership for
the non-root manager inside the named `credentials` volume. The runtime mounts
that volume read-only; only encrypted device TCP 9757 is published on the LAN.

Running setup again validates the same enrollment, permissions, token and TLS
pair without changing files. Different enrollment, incomplete credentials,
symlinks, incorrect ownership or expired TLS cause an error, not silent reset.
Files are prepared in a private staging directory and published together.
An interrupted run may leave a private `.setup-*` staging directory; it is not
used by the manager, and a retry creates a fresh staging directory.

## Connect a producer

The API remains `https://127.0.0.1:8443` inside the manager namespace. A trusted
adapter uses `network_mode: service:epaper-manager`, UID/GID 65532, and mounts
only the producer subdirectory (Compose must support volume `subpath`):

```yaml
services:
  adapter:
    image: YOUR_ADAPTER_IMAGE
    user: "65532:65532"
    network_mode: service:epaper-manager
    volumes:
      - type: volume
        source: credentials
        target: /run/epaper-api
        read_only: true
        volume:
          subpath: credentials/api
```

Use `/run/epaper-api/api-token` for Bearer authentication and
`/run/epaper-api/tls.crt` as the trusted certificate. Do not disable certificate
verification. The adapter receives neither the device key nor the TLS private
key; setup never prints tokens. Vikunja credentials are separate and remain
with your adapter. See [screen API](screen-api.md) for HTML submission and ACKs.

## Recovery and migration

The original [bind-mount Compose](container-deployment.md) remains supported;
existing installations need not migrate. This named-volume flow generates a
new API token/TLS pair only for a new destination, not a new device key.

Keep an operator-controlled encrypted backup of the volume and enrollment.
Do not use `docker compose down -v` as a troubleshooting step: it deletes the
credential volume. Manager restarts preserve credentials but require the
producer to resubmit its authoritative HTML.

Certificate renewal is not automatic: before its one-year expiry, replace the
TLS pair through your existing certificate-management process and redistribute
the new public certificate to producers. Preserve the API token and enrollment;
setup does not provide a rotation/renewal command.

## Sources

- [Docker named volumes and lifecycle](https://docs.docker.com/engine/storage/volumes/)
- [Compose volume subpath](https://docs.docker.com/reference/compose-file/services/#volumes)
- [Go x509 certificate creation and verification](https://pkg.go.dev/crypto/x509)
- [Go rooted filesystem operations](https://pkg.go.dev/os#Root)

## Portable ESP32 receiver (agreed target)

The same receiver must initiate an authenticated, encrypted connection from home
Wi-Fi or a foreign Wi-Fi network with a route to the manager. Support a stable
DNS name resolving to IPv4 and a literal IPv4 address. Moving the display must
preserve its USB enrollment and require no inbound port on the remote network.
Only the device listener may be exposed. Keep the HTML producer API private.

The v0.2.0 receiver uses authenticated EPN2 records over TCP. Publishing that
listener on port 443 does not provide TLS. The mobile TLS transport must verify
manager identity on every connection, reject failed verification without falling
back to TCP, and retain per-device authentication. A captive portal or a network
without a route to the manager can prevent delivery.

Acceptance requires confirmed frames using a DNS name and a literal IPv4 over
both LAN and an independent Internet connection, rejection of an untrusted
manager and unknown device, and recovery after switching networks without
re-enrolling. Public-network hardware acceptance is pending.

The mobile candidate uses a dedicated self-signed device-listener certificate and private key
through `-device-tls-cert` and `-device-tls-key` in Wi-Fi screen mode. With
both flags configured, its device listener accepts TLS 1.3 connections and
still requires EPN2 device authentication inside the session. The loopback
producer certificate serves a separate listener. Existing TCP receivers
cannot connect to the TLS listener.

USB binds the receiver to the manager certificate once. The mobile candidate
accepts replacement Wi-Fi credentials from a phone in physical setup mode
while preserving the device key and manager trust.

### Set up a mobile candidate

Choose a stable public DNS name for a traveling screen. Point it at the manager or a TCP forwarder, and forward the device port on the server side if needed. The visited Wi-Fi only needs outbound access. The device verifies its pinned TLS identity over this connection. This design requires no VPN peer, private tunnel address or public certificate purchase. For a new receiver, complete [first USB enrollment](../firmware/esp32/README.md#flash-and-first-time-provisioning) first.

Use matching firmware, provisioning CLI and manager builds. On the trusted
server that already has the private enrollment, generate the device listener
identity and owner setup card once. Include every DNS name or IPv4 literal the
screen will use. Dynamic public IPv4 installations can use a stable DNS name.

```sh
epaperprovision -registry /private/device.json mobile-init ./mobile paper.example.net 192.168.1.10
```

Keep the server key private. On Linux grant the manager's runtime user (65532)
read access to the two certificate files, for example with
`sudo chown 65532:65532 mobile/device-tls.{crt,key}` after copying the public
certificate to your USB provisioning host. Keep the phone setup card outside
the adapter and manager containers. Set `EPAPER_MOBILE_DIR` to the absolute
mobile directory, use a manager image built from this change, then add
`-f compose.mobile.yaml` after `-f compose.setup.yaml` in the Compose commands
above. For example, from the repository root with the settings in your environment:

```sh
docker compose -f deploy/compose.setup.yaml -f deploy/compose.mobile.yaml --profile setup run --rm setup
docker compose -f deploy/compose.setup.yaml -f deploy/compose.mobile.yaml up -d epaper-manager
```

The overlay publishes only raw device TLS on port 443 (override with
`EPAPER_DEVICE_PORT`). Use TCP passthrough if routing through a proxy. Ensure
the chosen address and port are reachable from the screen's network.

Install the matching application and bind an already enrolled screen over USB:

```sh
epaperprovision -target esp32 -port /dev/cu.YOUR_ESP32 set-manager paper.example.net device-tls.crt
```

This stores the address and SHA-256 digest of the self-signed P-256 leaf while
retaining the device ID and key. A replacement certificate needs one USB
rebind. The firmware requires the configured DNS/IP identity and pinned leaf.
The target SDK configuration has no wall-clock certificate date checking.
Keep the TLS key and existing enrollment available to a replacement manager.

**Change Wi-Fi from a phone.** Power up normally, then immediately hold BOOT
for about one second after the ROM has started. The firmware samples S1/BOOT
during the first three seconds of application startup. Holding BOOT while
applying power enters the ROM downloader. Use the network name and password
from the private `phone-setup.txt`, open `http://192.168.4.1/` on the phone,
and enter a WPA2-PSK/CCMP network name and password. This page runs on a
temporary password-protected access point with one client. Its two-minute
window is available only to an already enrolled TLS receiver. The panel keeps
its last image during setup. The receiver tests association and IPv4 acquisition
before saving only the Wi-Fi fields. On failure it restarts with the previous
network. The form submission confirms receipt; confirm delivery of a new
revision in the manager after reconnecting.

[Waveshare Rev3 board layout](https://files.waveshare.com/wiki/E-Paper-ESP32-Driver-Board/E-Paper_ESP32_Driver_Board_V3.pdf)

A TLS-bound receiver rejects a USB endpoint-only change to legacy TCP. To
change addresses while keeping trust, run `set-manager NEW_HOST device-tls.crt`
with a name or address covered by that certificate.

Local candidate verification on 2026-11-19 passed `make quality` and
`make latency` in the ESP32 Dev Container on a source snapshot, excluding the
operator's ignored host flasher environment. The three Go coverage runs each
reported 94.8% overall, with 93.1% on changed statements. The Linux test runs
the firmware's C TLS client against Go using a certificate generated by
`epaperprovision mobile-init`. It exchanges data for DNS and IPv4 and rejects
a different certificate digest, hostname mismatch and TLS 1.2. These checks
did not exercise a real phone, external network or ESP32 radio.
