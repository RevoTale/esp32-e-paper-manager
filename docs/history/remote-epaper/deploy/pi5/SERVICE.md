# Local e-paper service on Raspberry Pi 5

An ordinary Go Linux ARM64 process, not a UF2. It targets the **Waveshare
2.13inch e-Paper HAT Rev2.1 with V4 panel**, not the separate 7.5-inch HAT.
Existing Pico firmware is unchanged.

## What is ready, and what is not

Software: HTML/PNG conversion, native 122×250 frame, default partial-session driver,
SPI/GPIO adapter, authenticated Unix HTTP, lifecycle tests and ARM64 build.
There is no simulated backend or TCP listener in the shipped executable.

User-reported target checkpoint,2026-09-08: real DTB trial merge succeeded,
SPI5 appeared after reboot, Pironman appeared normal, the Unix API responded,
and the full HTML test was visibly displayed. On2026-09-09 the user also
visually confirmed partial, five-partial/full cleanup and idle recovery.
Systemd/Docker deployment is not yet target-accepted. The offline DTB fixture
is **not** the installed Pi DTB; a build alone never establishes wiring safety.

The configuration uses SPI5 TX-only GPIO14/15 with kernel CS16 and control
GPIO22/23/24 (DC/reset/BUSY). The full-frame checkpoint used the eight-wire map
recorded in `docs/epaper-debugging-history.md` at the repository root. Never
stack it on Pironman's full header or leave Pico attached. Do not disable
RGB/OLED/fans to make this service work. Power down before changing wires.

## Build in the existing Dev Container

From `experiments/03-remote-epaper`:

```sh
sh scripts/quality.sh task
go test -race ./...
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o build/epaper-local ./cmd/epaper-local
dtc -@ -I dts -O dtb -o build/epaper-spi5-pi5.dtbo deploy/pi5/epaper-spi5-pi5-overlay.dts
sha256sum build/epaper-local build/epaper-spi5-pi5.dtbo
```

The existing `engine` supplies the bounded inline HTML/CSS profile. `<style>`,
JavaScript, network fetching and browser compatibility are not promised.
See `SPEC-engine.md` in the module/package root for the accepted profile; the
older manager HTML/CSS profile is historical, not the active contract.

## Stage on the Pi — only when ready to install

For the one-command package installation, use [the Debian package guide](deb/README.md).
The commands below are the alternative manual installation; do not mix its
`/etc` unit with the package-owned `/usr/lib` unit.

These commands are operator instructions; they have not been run on your Pi.
From the extracted package/repository root containing `build` and `deploy/pi5`:

```sh
sudo install -m 0755 build/epaper-local /usr/local/bin/epaper-local
sudo install -m 0644 deploy/pi5/epaper-local.sysusers /etc/sysusers.d/epaper-local.conf
sudo systemd-sysusers /etc/sysusers.d/epaper-local.conf
sudo install -d -m 0700 /etc/epaper-local
sudo sh -c 'umask 077; set -C; openssl rand -hex 32 > /etc/epaper-local/token'
sudo install -m 0644 deploy/pi5/epaper-local.service /etc/systemd/system/epaper-local.service
sudo systemd-analyze verify /etc/systemd/system/epaper-local.service
sudo systemctl daemon-reload
```

Token generation refuses to overwrite an existing file. The random token is
never a command-line argument or printed to the terminal. OpenSSL is an
operator-side provisioning tool, not a service dependency. The service accepts
a private regular file with 32..256 non-whitespace characters and optional final
newline; it refuses symlinks and group/world-readable credentials.

Do **not** enable/start yet if `/dev/spidev5.0` is absent. Follow the overlay
README's real-DTB verification first; stock `spi5-1cs` claims unsuitable pins.
Install the custom overlay and edit `/boot/firmware/config.txt` only after
reviewing those checks; a reboot and renewed pin-ownership check are required.
Existing SPI0 and SPI10 must remain unchanged.

The reported kernel was `6.18.39+rpt-rpi-2712`; gpiochip4 was an alias of
gpiochip0. The executable verifies the selected chip label (`pinctrl-rp1`) and
line names rather than trusting the number. If the chip path changes, update
**both** `-gpiochip` and `DeviceAllow` in the unit before starting.

After electrical/overlay acceptance:

```sh
sudo systemctl enable --now epaper-local.service
sudo journalctl -u epaper-local.service -n 50 --no-pager
getent group epaper-api
```

Startup does not refresh. In full-only mode wait at least 180 seconds before the first submission;
the startup floor also protects restart recovery. A missing/occupied device is
an explicit startup failure, not a fallback to another GPIO/SPI bus.

## Frequent status updates (V4 partial sessions)

Partial sessions are the default. Use `-partial=false` for full-only fallback:

```sh
~/epaper-local-live -token-file ~/epaper-local-run/token \
  -socket "$HOME/epaper-local-run/api.sock"
```

Stop the old foreground instance with Ctrl+C before starting the new one.
The lock prevents two processes owning the same socket. Do not run both against
the same display under different socket paths. No new token or overlay needed.
The existing HTTP HTML/PNG requests are unchanged. Nothing captures the shell
automatically; a producer must submit rendered text/HTML/PNG snapshots.

| Option | Partial-mode default / constraint |
| --- | --- |
| `-refresh-interval` | 1s after physical completion; at least 1s |
| `-max-partial` | 5; may be reduced to 1..4, never increased above 5 |
| `-full-refresh-interval` | 10m; full on next changed frame when base is this old |
| `-idle-sleep` | 30s without physical updates, checked every second |

Full/idle intervals must be at least the configured refresh interval. Live-only
options with `-partial=false` are rejected, not silently ignored. The first live
frame is accepted immediately and performs full base initialization. Up to five
changed frames then use the partial waveform; the next is full. This count guard
can trigger cleanup well before ten minutes. The V4 specification p9 recommends
full after five partial/fast operations; this is not a continuous-lifetime test.

Each partial still uploads a complete 4000-byte bitmap, like the official V4
driver; no cropped-window bandwidth claim. No waveform, SPI-speed or temperature
overrides. BUSY decides completion; one second is a scheduling gap, not promised
end-to-end refresh latency. A full cleaning cycle can still visibly flicker.

During the active burst the controller retains its base RAM. Idle/graceful-stop
sleep invalidates that base; the next submission is full, with no three-minute
startup cooldown. Hardware failure also requires full recovery, without blind
retries. A sleep failure terminates the process rather than continuing unknown
state. The clock corner stays fixed during partial updates and changes on full.
Unchanged frames neither refresh nor prolong the awake period. Busy/cooldown
still returns 429 with Retry-After: submit the newest snapshot later, not a
backlog of old terminal states. No automatic replays after ambiguous responses.

Physical partial/cleanup/sleep-wake confirmation is required before accepting
this mode. See repository `docs/pi5-service/SPEC-live-refresh.md` for sources,
state rules and tests. For rollback stop the live process and run preserved
`epaper-local-full-baseline` without `-partial` (full-only 180s floor).
With the new binary, use `-partial=false` instead. Replacing a systemd binary
also changes its default mode unless ExecStart explicitly sets `-partial=false`.

## Docker client and credential

Merge `compose.client.yaml` into the existing application. Set its existing
image, numeric UID/GID and the host `epaper-api` numeric GID. No new container
has been started by this work. Rootless/user-namespace Docker needs a separate
UID/GID mapping check; do not solve a mismatch by making the socket world-writable.

The mounted file `/run/secrets/epaper-authorization` contains one complete line:
`Authorization: Bearer <the same token>`, **not** a bare token. Create it without
echoing the token; replace `1000:1000` below with the actual application UID/GID:

```sh
sudo sh -c 'umask 077; set -C; { printf "Authorization: Bearer "; cat /etc/epaper-local/token; } > /etc/epaper-local/client-authorization'
sudo chown 1000:1000 /etc/epaper-local/client-authorization
sudo chmod 0400 /etc/epaper-local/client-authorization
```

For an application container that already has curl:

```sh
curl --fail-with-body --unix-socket /run/epaper-local/api.sock \
  -H @/run/secrets/epaper-authorization http://unix/v1/status
curl --fail-with-body --max-time 120 --unix-socket /run/epaper-local/api.sock \
  -H @/run/secrets/epaper-authorization -H 'Content-Type: text/html; charset=utf-8' \
  --data-binary @dashboard.html http://unix/v1/frame
curl --fail-with-body --max-time 120 --unix-socket /run/epaper-local/api.sock \
  -H @/run/secrets/epaper-authorization -H 'Content-Type: image/png' \
  --data-binary @frame.png http://unix/v1/frame
```

The token is read from a file, not expanded into curl's argument list. Treat
both credential files as secrets; do not add them to the repository or logs.
On rotation, replace both privately, restart the host service, and remount the
client credential if its bind-mounted inode was replaced. The socket directory
is preserved across service stop/start, so socket replacement alone does not
require restarting the container. Reconnect the HTTP client; never blindly
retry a POST after an ambiguous disconnect—check status first.

## API and operating policy

Every endpoint needs the bearer token. Socket mode is 0660 inside a 0750
directory, owned by the service and `epaper-api` group. A read-only Docker bind
does not prevent socket requests: the token and filesystem group both matter.

| Request/result | Meaning |
| --- | --- |
| `GET /v1/status` | Capabilities, phase, latest admitted attempt, next allowed time |
| `POST /v1/frame` | Complete HTML (≤32 KiB) or PNG (≤1 MiB encoded, ≤1 MP decoded) |
| `200 controller-complete` | Driver completed; full-only also slept, live retains RAM; not optical proof |
| `200 unchanged` | Same unstamped pixels; no hardware I/O and no timestamp change |
| `401 / 413 / 415 / 422` | Unauthorized / too large / unsupported MIME / invalid content |
| `429` + `Retry-After` | Busy or cooldown; no queued update |
| `408 / 503` | Cancelled before hardware / stopped or device failure |

Each admitted attempt has an ID; IDs restart with the process. Rejections are
source-free. Hardware failure status adds stage, command, row offset and last
known BUSY state when supplied by the controller driver. `busy_known=false`
means no electrical level was established, not LOW. Logs contain no input or
credentials. Status after a restart cannot establish what the old panel shows.

The logical viewport is 250×122 landscape, rotated once clockwise. PNG is
contained on white with alpha compositing and deterministic Bayer quantization.
The bottom-right 120×21 rectangle is reserved and cleared before stamping.
Its date/time is the **start** of the last completed full-refresh cycle, not a
future completion time. Zone defaults to Europe/Kiev; override `-timezone` in
the unit. Full-only mode retains its 180s interval after completion or process
start; `-partial` uses the separate policy above. The old floor is conservative
project policy, not an intrinsic three-minute hardware operation. No periodic
clock repaint. Results/logs include `refresh_mode`/`mode`: full or partial.

After client disconnect before complete validation, no hardware operation is
started. After hardware begins, the service finishes independently of that
client. BUSY waits have a 20s per-stage bound; a stalled Linux syscall is owned
by the kernel and cannot be interrupted by that Go deadline. Shutdown stops
admission and waits up to 90s; systemd has a 100s stop limit. Do not unplug or
rewire during an update. Failed updates are not retried automatically.

## Acceptance and rollback

Confirm distinct HTML and PNG pixels manually, then check retained Pironman
RGB, OLED and fans through refresh, idle and service restart. Record binary
SHA-256, kernel, real overlay/pins, wiring, API status and your visible result.
The user confirmed the first full HTML frame; partial/idle-wake and systemd/
Docker acceptance remain pending. No remote hardware operations were performed
by the agent during partial implementation.

Rollback: `sudo systemctl disable --now epaper-local.service`. Keep credentials
private; do not delete unrelated files or change the Pico firmware. Restore only
the custom overlay/config entry added for this service, reboot, then check
Pironman's functions. Do not replace the whole boot configuration.

Configuration sources: [systemd execution/credentials](https://github.com/systemd/systemd/blob/v257/man/systemd.exec.xml),
[device allow-list](https://github.com/systemd/systemd/blob/v257/man/systemd.resource-control.xml),
[Unix sockets](https://man7.org/linux/man-pages/man7/unix.7.html).
