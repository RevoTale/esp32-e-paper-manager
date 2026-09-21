# Spec: debian-deployment

Status: executable and configuration implemented; physical/systemd acceptance
on the Pi remains pending. Operator guide: module `deploy/pi5/SERVICE.md`.

## Objective and artifacts

Provide reviewed Debian Trixie systemd configuration and Docker client usage
without replacing the Pironman service. Proposed files live under
`experiments/03-remote-epaper/deploy/pi5/` with an operator README.
Use the ordinary Go ARM64 executable; do not flash a UF2 onto Raspberry Pi 5.

## Configuration

- Dedicated unprivileged service account, private credential and runtime
  directory. Explicit access only to selected SPI and GPIO character devices.
- systemd hardening must permit required device I/O; do not combine an
  inaccessible private /dev with a claim that GPIO works. Verify the unit.
- Docker clients get a bind mount of the API socket directory and matching
  group access. No privileged container, Docker socket or GPIO passthrough.
- Document token delivery without command-line token literals, shell history
  leakage or tracked secrets. No publicly bound unauthenticated port.
- Supply the reviewed custom overlay separately. Keep SPI0/Pironman settings;
  clearly explain installation, reboot and rollback. Installation is a manual
  approval boundary, not an action implied by writing config files.
- Publish the final eight-wire HAT-to-Pi mapping with BCM and physical pin
  numbers only after schematic and overlay verification. Disconnect all power
  before wiring; never retain Pico connections or stack this HAT directly.

## Verification

Validate unit syntax with `systemd-analyze verify` where available, cross-build
ARM64, and run the service API in the existing development environment using a
fake device. A fake backend must be explicit and cannot claim physical success.
Use an existing authorized Docker environment for end-to-end socket tests;
ask before creating or starting any new container.

Physical acceptance on target: check device permissions, boot overlay and pin
consumers; submit HTML and PNG from Docker; observe distinct pixels and retained
Pironman OLED, RGB and fans; restart service and confirm safe rate limiting.
Record exact binary hash, OS/kernel, wiring, API result and observed image.

Done requires all four module acceptance gates. No commit, push, homelab
installation, reboot or hardware action without the relevant user authority.
