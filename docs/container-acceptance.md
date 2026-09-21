# Container acceptance — 2026-09-21

This records local packaging evidence, not production or physical acceptance.
The operator authorized image builds and temporary isolated containers. Existing
Dev Container, macOS manager, ESP32 firmware and real credentials were untouched.

## Runtime image

The first independent build failed because the official TinyGo build image uses
a non-root user and could not create `/out`. The build now writes its binary to
`/tmp/epaper-manager`; the final runtime still uses UID/GID 65532. Rebuilding the
actual Dockerfile is the regression check, also required by the quality workflow.

Linux arm64 and amd64 runtime images built successfully. Arm64 local image size
reported by Docker: 44,712,533 bytes. This is not registry download size.

An additional disposable test image supplied synthetic enrollment, a dummy API
token, an independently generated one-day TLS certificate and an HTTP probe.
Those fixtures were not added to the production image or version control.

Both arm64 and amd64 managers ran with `--network none`, no published ports or devices,
read-only root, all capabilities dropped, no-new-privileges, 128 PID limit and
128 MiB memory limit. Checks returned:

| Check | Observed result |
| --- | --- |
| GET `/v2/screen/status`, missing bearer token | 401 |
| Same request, wrong token | 401 |
| Same request, matching token | 200 |
| TLS certificate chain and IP SAN validation | successful, verification enabled |
| Effective process UID | 65532 |
| Attempted root-filesystem write | rejected |
| Device state | current/in-flight/confirmed/delivered all 0; refresh untrusted |

No device was connected; the zero state is expected, not a display failure.
Both temporary runtime containers were stopped and removed after verification.
The two fixture image tags were also removed; production-candidate images remain
locally cached. No published registry was modified.

The amd64 `-help` command starts and prints usage, but the existing CLI returns
exit status 1 for `flag.ErrHelp`. Do not use `-help` as a successful health check.
The authenticated API test above, not that exit status, proves runtime startup.

## Independent development image

The first build reached ESP-IDF installation but OpenOCD could not load
`libusb-1.0.so.0`. Added Debian package `libusb-1.0-0` explicitly to the image.
The SDK install and `idf.py --version` build step must succeed; this dependency
must not be supplied accidentally by a pre-existing development container.
The corrected image built successfully and its build-time SDK check reported
`ESP-IDF v5.5.5`.

A disposable, network-disabled container from that image also ran Go 1.26.8,
TinyGo 0.42.0, ESP-IDF 5.5.5 and OpenOCD v0.12.0-esp32-20260424 successfully.
An initial `bash -lc` probe failed with `go: command not found`: the login
profile replaces PATH. `bash -c` preserves the image environment and passes.
This is recorded in CONTRIBUTING.md; VS Code interactive-shell acceptance has
not been performed.

## Boundaries still open

- GHCR publication and real GitHub workflow execution.
- Compose bind-mounted secrets ownership/permissions on the deployment host.
- Hardware delivery, recovery and rollback from the packaged manager.
- No claim that local image tags identify a published release.

## Clean-source qualification follow-up

The new development image ran `make quality` against an isolated source archive
in `/tmp/project`, with a fresh clone of the destination repository as its Git
baseline. The archive contained only Git-listed tracked/unignored files; build
outputs, credentials and caches were excluded. The container could not access
the source worktree except for that read-only archive. No commit was created.

The complete snapshot gate passed: formatting, all three Go-module lint runs,
race tests, vet, module verification, coverage, 18 native C tests, actual Go/C
interoperability, three TinyGo test builds, ESP32 application build and both
Linux manager architectures. The ESP32 binary remained 772400 bytes with 81%
application partition free; equal size does not imply identical binary hashes.

During that run, the working tree separately gained pinned actionlint 1.7.7 and
mandatory `make workflows` / `make audit` targets. Both targets passed in the
existing development container; govulncheck reported no known vulnerabilities
for root Go packages on this date. These extra targets were not in the already
running clean snapshot. This is not an audit of ESP-IDF or Debian packages.

README formerly pointed readers at the historical Blitz/EPS1 guide. It now
points to the code-checked `screen-api.md`; original historical material stays
available. The source manifest now lists 730 destination files, but remains a
provisional provenance map rather than a completed audit of every old link.

Temporary smoke fixtures were removed after use, including synthetic keys and
the local probe program. A format check correctly caught the temporary probe
under `build/`: Go package discovery does not respect Git ignore rules. Removing
the finished test artifacts restored the format gate; no checker was weakened.
