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

## GitHub CI checkpoint

Commit `37f932f03300fc45fe2318a5eb8ae70180007fe8` passed
[Quality run 35649607308](https://github.com/RevoTale/esp32-e-paper-manager/actions/runs/35649607308)
on branch `codex/standalone-manager-ci` at 2026-09-21 20:21:49 UTC. GitHub built
the independent toolchain, ran the complete `make quality` including workflow
lint and Go vulnerability checks, and built the amd64 runtime image. No merge,
release tag, registry publication or new physical delivery was performed.

The subsequent documentation audit found all 332 unique external URL strings
from manifest-mapped source Markdown in their copied documents. This comparison
does not prove remote links are reachable or that the manifest contains every
relevant source file. Broken relative links to migrated specifications/history
were retargeted without changing the historical conclusions. Old local Pico
build artifacts and the package README template retain explicitly labelled
historical paths; they are not standalone release downloads.

A broader tracked Markdown/PDF inventory then found five documents outside the
initial map: the Blitz probe README, three Pi5 deployment/packaging guides and
the 1.54-inch panel source notes. These were copied byte-for-byte into history
and added to the manifest. Their implementations remain in the parent project.
This closes that inventory gap without representing those platforms as current
ESP32 runtime dependencies.

## Native C policy follow-up (local, not committed)

Added clang-tidy 19 to the development Dockerfile and a 60-line function policy.
The installed development-container tool passed the scan of 39 native-tested C
files. A CTest policy probe accepts a short function and rejects a generated
61-statement multiline function using the actual project config; five bounded
repetitions passed. This covers native core/tests and net_io/display_guard/uart,
not the remaining ESP-IDF-only adapters. The changed image has not been rebuilt.

The integrated native run is **not green**: `net_io_fault.shared_deadline` failed
the saturated-write wall-clock assertion. Bounded repetitions reproduced it;
added failure-only diagnostics measured 123259 microseconds for a 50000-us
deadline against the unchanged 120000-us test ceiling. The read assertion was
not the failing assertion. Container CPU quota was unlimited, cgroup throttling
counters were zero and the shell timer slack was 50000 ns. Scheduler/VM latency
is a hypothesis, not a demonstrated cause. No deadline, assertion, skip or
firmware behavior was changed. Resolve this before treating the new full gate
as passing; the earlier GitHub success applies only to commit `37f932f`.

Follow-up localization used a temporary native linker wrapper around `select`,
preserving its arguments, return value and errno. Calls requesting 25000 us
took 76717–119796 us wall time with 28–104 us thread CPU time; the failed write
scenario measured 153274 us. The delay is inside system waiting, not an observed
deadline reset in the application. Linux documents that scheduling may overrun
the requested select timeout: https://man7.org/linux/man-pages/man2/select.2.html.
The exact host/VM scheduling cause remains unknown. The temporary wrapper was
removed after collecting evidence; failure-only elapsed diagnostics remain.

Proposed follow-up needs explicit quality-contract agreement: test deadline
non-renewal and cancellation deterministically with a controlled clock/socket
wait, keep real-socket fragmentation/EOF/backpressure coverage, and separate
wall-clock latency qualification from the ordinary shared-container unit gate.
Do not silently delete the current latency assertions or raise their limits.

An additive `net_deadline` native test now runs the unchanged production
`net_io.c` with linker-wrapped socket operations and a controlled clock. It
covers fragmented reads/writes, EINTR, EAGAIN, cancellation, readiness after
deadline expiry, normal completion, EOF and an already-expired deadline. The
test and its C function-size check passed. A build-directory-only mutation
that renewed the deadline on every transfer iteration failed the bounded wait
assertion, demonstrating that the test detects the intended regression.
No production source changed; the original real-socket test and its latency
ceilings remain mandatory while the proposed contract change awaits approval.
The subsequent complete native run passed 20/20 tests in 1.08 seconds. Since
the production implementation and old latency limits did not change, this pass
does not resolve the previously reproduced environment-sensitive latency failure.

The next complete `make quality` run exited 2 at `make native`: 19/20 native
tests passed, including `net_deadline`; the original shared-read wall-clock
assertion measured 120587 us against the unchanged 120000-us ceiling. Earlier
localization found a write failure; this later run establishes that the read
ceiling is also environment-sensitive. Upstream gates through Go lint, workflow
lint, vulnerability audit, Go tests and coverage completed before this failure.
Later C-size/interoperability/TinyGo/firmware/runtime build targets were not
reached in this run. Do not use the older successful CI or isolated 20/20 run
as evidence that the current complete gate is stable.

Continuation now requires a decision on the previously proposed separation of
deterministic correctness and wall-clock qualification. No assertion has been
removed. Release remains additionally pending dependency-notice resolution and
separate publication authority; no upstream contact, merge or publish occurred.

## 2026-09-26: approved latency qualification separation

The user approved separating wall-clock qualification and declined creating an
upstream scanx issue. The earlier pending-decision entries remain historical.
No upstream issue was created; the dependency-notice blocker remains unresolved.

`make native` retains all 20 correctness tests, including real socket I/O and
the controlled-clock deadline regression test. `make latency` builds the same
socket source with explicit latency assertions and reports each measurement.
The original 120000-us read/write and 150000-us cancellation ceilings remain.
Both socket executables have a 15-second test-runner safety timeout. Production
network code and firmware behavior are unchanged.

Measured in the existing `e2b84686dae7` Dev Container: native 20/20 passed;
separate latency 1/1 passed (read 74821 us, write 55040 us, cancellation 26844 us);
clang-tidy C function-size check passed. This is one host qualification sample,
not proof of ESP32 timing or that host scheduling overruns can never recur.

The subsequent full `make quality` exited 0, including format/lint, workflow
checks, vulnerability audit, Go tests/coverage, native tests, C function-size,
Go/C interoperability, TinyGo package compilation, ESP-IDF firmware build and
CGO-free Linux amd64/arm64 manager builds. The ESP32 application is 0xbc930
bytes with 81% app-partition space free. Log: local ignored
`build/quality-20260926.log`. This worktree has not yet had a new GitHub CI run;
the Dev Container image was not rebuilt and no device was flashed.
