# Local ARM64 Debian package

For the already verified Pi 5 + 2.13-inch V4 SPI5 wiring only. No overlay,
boot configuration, Pironman service, GPIO setting or credential is packaged.
Packaging copies the supplied executable and validates its ELF architecture.
Version .2 fixes native systemd credentials; the panel driver is unchanged.
Use `BINARY=build/epaper-local-systemd` for that corrected candidate; preserve
the optically accepted standalone `build/epaper-local-live` for rollback.

## Build

Inside the existing Dev Container, from the module root:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o build/epaper-local-systemd ./cmd/epaper-local
make -f deploy/pi5/deb/Makefile package BINARY=build/epaper-local-systemd
dpkg-deb --info build/epaper-local_arm64.deb
sha256sum build/epaper-local_arm64.deb
```

Uses Debian debhelper (compat13), including dh_installsystemd and
dh_installsysusers. Bump `debian/changelog` for each changed package.
OUTPUT and BINARY can be overridden;
SOURCE_DATE_EPOCH clamps archive timestamps for repeatable packaging.

## First installation on the Pi

```sh
sudo apt install ./epaper-local_arm64.deb
```

Installation creates the service identity; it neither starts nor enables the
service. A local unit in `/etc/systemd/system/epaper-local.service` or sysusers
file in `/etc/sysusers.d/epaper-local.conf` causes an early refusal: preserve
and explicitly migrate it first, rather than silently overriding local policy.
An old `/usr/local/bin/epaper-local` is untouched and may still shadow shell PATH;
the packaged unit explicitly uses `/usr/bin/epaper-local`.

One-time credential migration (only if no production token already exists):

```sh
sudo install -d -m 0700 /etc/epaper-local
sudo test ! -e /etc/epaper-local/token && sudo install -m 0600 ~/epaper-local-run/token /etc/epaper-local/token
```

Stop the foreground process with Ctrl+C before enabling the system service.
Do not run both: the old home socket and new system socket have different locks.

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now epaper-local.service
sudo journalctl -u epaper-local.service -n 20 --no-pager
```

API socket is now `/run/epaper-local/api.sock`. Use a root client for initial
verification; non-root clients require membership in epaper-api and their own
secure token provisioning. No TCP listener or firewall opening is added.

## Update and rollback

Debhelper generates account creation, enable-state/purge management and manager
reload hooks. `--no-enable --no-start` retains deliberate operator activation.
One removal-stop hook remains because compat13 omits it with `--no-start`;
it prevents leaving a hardware owner running after package removal.

APT owns `/usr/bin/epaper-local` and `/usr/lib` unit/sysusers files, never
`/usr/local`. Upgrades stop the managed service but intentionally do not restart
it: operator runs `sudo systemctl start epaper-local` after installation.
Remove stops the service; credentials and account are retained even on purge.
Keep the prior .deb to install explicitly for rollback. The preserved standalone
full-only binary remains another fallback after stopping the system service.

## Evidence and limits

Archive/maintainer-script tests do not prove real systemd startup, boot recovery
or Debian dependency resolution on the Pi. Verify those on the target. Never
install this hardware package into the development container to test GPIO.

The service consumes `token` from CREDENTIALS_DIRECTORY. Systemd owns ACL and
directory isolation; the application checks regular-file type, no final symlink,
bounded length and token format. Explicit `-token-file` retains strict private
permissions and does not use the systemd policy. Never chmod runtime credentials.
The launcher environment is trusted configuration, not input from HTTP clients.

After updating on a systemd host, this exercises real LoadCredential without
touching GPIO or the API socket (no token value is printed):

```sh
sudo systemd-run --wait --pipe --collect -p User=epaper-local -p Group=epaper-api -p ProtectSystem=strict -p LoadCredential=token:/etc/epaper-local/token /usr/bin/epaper-local -check-credential
```

Expected: exit0. The existing Dev Container runs sh as PID1, so this integration
check cannot run there; systemd tools installed there do not change that fact.

Sources:
- https://www.debian.org/doc/debian-policy/ch-opersys.html (no package files in /usr/local)
- https://www.debian.org/doc/debian-policy/ch-maintainerscripts.html (idempotent lifecycle)
- https://manpages.debian.org/trixie/dpkg/dpkg-deb.1.en.html (archive validation, root ownership)
- https://manpages.debian.org/trixie/init-system-helpers/deb-systemd-invoke.1p.en.html (policy-aware service stop)
- https://systemd.io/CREDENTIALS/#programming-interface-from-service-code
- https://github.com/systemd/systemd/issues/29435 (ACL-mask compatibility)
- https://manpages.debian.org/trixie/debhelper/dh_installsystemd.1.en.html
- https://manpages.debian.org/trixie/debhelper/dh_installsysusers.1.en.html
