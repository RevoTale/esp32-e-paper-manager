# Simple Docker setup

Prepare manager credentials once, then start the manager normally. No manual
certificate generation, token generation, UID changes or chmod on generated
files. ESP32 firmware and USB provisioning are separate steps.

Requires Docker Compose, an existing private USB enrollment file, and a manager
image containing `epaper-manager setup`. **v0.1.1 does not contain this command.**
Until this change is published, build the candidate image explicitly; do not
invent a release version or assume `latest` includes it.

## Install

Copy `deploy/compose.setup.yaml` to your deployment directory as `compose.yaml`.
Put these three non-secret values in `.env` next to it:

```dotenv
EPAPER_IMAGE=ghcr.io/revotale/esp32-e-paper-manager@sha256:REPLACE_WITH_RELEASE_DIGEST
EPAPER_ENROLLMENT_FILE=/absolute/path/to/private/device.json
EPAPER_DEVICE_BIND=192.168.1.10
```

The LAN address must match the manager address already provisioned on ESP32.
Changing Compose does not change that device setting. The enrollment source
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
