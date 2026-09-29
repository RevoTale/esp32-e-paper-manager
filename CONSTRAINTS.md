# Quality contract

Reviewed 2026-09-21. This standalone contract supersedes historical paths, not
the approved quality bar. The unchanged parent contract is archived in
[CONSTRAINTS-parent.md](docs/history/CONSTRAINTS-parent.md). Historical waivers
do not authorize skips or suppressions here.

## Non-negotiable rules

- Never track credentials, private HTML captures or enrollment state.
- No new lint suppression, skipped test, deleted assertion, coverage exclusion,
  empty error handling or stub merely to obtain a green gate.
- Preserve protocol vectors and error/recovery behavior during extraction.
- Any generated/vendor/hardware-only exclusion needs an exact path, rationale,
  owner, expiry and explicit approval. Never silently lower this contract.
- Builds and host tests do not prove physical refresh or production acceptance.

## Enforced gates

| Rule | Enforcement |
| --- | --- |
| At most 300 physical lines per owned Go/C/header file | `make format` |
| At most 60 lines per Go function, complexity at most 10 | pinned golangci-lint, funlen and cyclop |
| At most 60 lines per native-tested C function | clang-tidy 19 readability-function-size, `make c-size` |
| Changed executable coverage at least 90% | worktree-covercheck, including untracked implementation |
| Root Go statement coverage at least 94.6% | `make coverage`, Sep 21 no-decrease baseline |
| Stable covered blocks across three independent runs | `make coverage`, uncached set-mode profiles compared after sorting |
| Whole Go modules linted, not only changed files | root, tools and firmware interoperability modules |
| GitHub workflow syntax and contracts | pinned actionlint, `make workflows` |
| Known reachable Go vulnerabilities | pinned govulncheck, `make audit` (live Go vulnerability database) |
| Race tests, vet and module verification | `make test` |
| Native receiver tests and Go/C interoperability | `make native interop` |
| Portable packages compile with TinyGo | `make tinygo`, Pico 2 W test executables |
| Real ESP32 application builds | `make firmware`, ESP-IDF 5.5.5 |
| Linux amd64/arm64 manager builds without CGO | `make build` |

Run `make quality` for the complete gate. `QUALITY_BASE_REF` selects the review
baseline; CI uses the PR base or previous pushed commit. A clean-worktree result
is not evidence for a different unreviewed revision.

TinyGo compilation covers screenwire, streamrx and refreshpolicy, not a complete
Pico application or heap/stack budget proof. C function-length checking currently
covers core, native tests and the three native-tested platform files, not all
ESP-IDF-only adapters. Those adapters and complete target resource acceptance remain migration work,
not implied guarantees of this gate.

## Runtime and acceptance

Keep bounded input, layout, decompression and transport limits in their owning
modules and boundary tests. Never weaken controller safety or replay protection
for faster refresh. Rendering and physical-refresh durations are separate metrics.

The inherited five-second software-rendering target remains an acceptance
target, excluding physical refresh. The historical Pico watchdog applies to
Pico, not automatically to ESP-IDF. Unmeasured targets are not achieved results.
Fast local validation should fit 90 seconds; full firmware/CI and physical
checks are separate and must never be silently skipped or interrupted and then
reported as successful.

## Sources

- [golangci-lint settings](https://golangci-lint.run/docs/linters/configuration/)
- [Go coverage](https://go.dev/doc/build-cover)
- [Clang 19 function-size policy](https://releases.llvm.org/19.1.0/tools/clang/tools/extra/docs/clang-tidy/checks/readability/function-size.html)
- [covercheck v0.2.0](https://pkg.go.dev/github.com/dr-dobermann/covercheck@v0.2.0)
- [TinyGo allocations](https://tinygo.org/docs/concepts/compiler-internals/heap-allocation/)

## Approved latency qualification split

User-approved on 2026-09-21; implemented on 2026-09-26. `make quality`
retains deterministic deadline/cancellation checks and real-socket functional
tests. Only wall-clock ceilings run separately with `make latency`: shared
50 ms read/write deadlines must return in less than 120 ms, cancellation in
less than 150 ms. Both executables compile the same socket test source.

Measured host `select` scheduling overruns motivated this split; no firmware
deadline or latency threshold changes. See [select(2)](https://man7.org/linux/man-pages/man2/select.2.html).
Passing quality does not establish latency qualification. Maintainers must
record a separate controlled-environment latency result before production
acceptance; review this split when the qualification environment changes.
