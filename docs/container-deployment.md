# Container deployment

Status: packaging candidate; image build/start and registry publication have
not yet been accepted. Do not treat a proposed image tag as a published image.

The image runs the Go manager as UID/GID 65532. The ESP32 firmware remains on
the board. Only encrypted device-link TCP 9757 is published, on an explicitly
selected LAN address. Keep that port behind the LAN firewall by default.

## Credentials

Create credentials with the documented USB provisioning flow; do not invent an
enrollment JSON or rotate an existing device key during deployment. Store
`device.json`, `api-token`, `tls.crt`, and `tls.key` in a private directory outside
the checkout/build context. On Linux, private files must be readable by UID
65532 with mode 0600; the directory needs traversal permission for that UID.
Do not make credentials world-readable to work around ownership errors.

Configure these Compose interpolation variables:

- `EPAPER_IMAGE`: verified `ghcr.io/revotale/esp32-e-paper-manager@sha256:…`.
- `EPAPER_DEVICE_BIND`: the server's LAN IP, reachable from the ESP32.
- `EPAPER_SECRETS_DIR`: absolute path to the credential directory.

Inspect configuration with `docker compose -f deploy/compose.yaml config`.
Starting it is a separate operator action:
`docker compose -f deploy/compose.yaml up -d`.

## API isolation

The current screen API deliberately binds to 127.0.0.1 inside the container.
Publishing port 8443 will NOT make it reachable. A trusted producer container
can share its network namespace with `network_mode: service:epaper-manager`
in the same Compose project, then call `https://127.0.0.1:8443/v2/screen`.
That producer still needs the API token and CA certificate. Do not expose it
through an unauthenticated proxy or weaken the renderer's trust boundary.

Producer integration is application-specific and is not a bundled Vikunja
adapter. See [the screen API](screen-manager-usb.md) for If-Match, acceptance
versus delivery status, and supported HTML/CSS.

## State and recovery

Enrollment and keys live in the mounted secret directory. Screen HTML is held
in memory; after a manager restart the producer must submit the current scene
again. The existing display may retain old pixels; this does not prove that
the restarted manager confirmed them. Retain operator backups of enrollment
and firmware; never delete them as a retry mechanism.

Readiness requires an authenticated status request, not merely an open TCP
port. Hardware acceptance additionally requires a confirmed delivery and a
visible new frame. No Docker HEALTHCHECK is claimed in this candidate.

Rollback: restore the previously accepted image digest while retaining the
same credentials, then resubmit the scene from its authoritative producer.
Do not roll back across incompatible wire or enrollment versions silently.
