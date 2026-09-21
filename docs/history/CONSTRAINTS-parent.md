# Constraints

Status: accepted quality bar; V2-01 mechanically enforces the bootstrap rules.

Last reviewed: 2026-09-02.

## Scope

These rules apply to all new or modified code in this repository. The remote
HTML e-paper firmware additionally follows
`experiments/03-remote-epaper/REQUIREMENTS-v2.md`.

## Floor

- No secrets or private source captures in tracked files.
- No new `//nolint`, coverage-ignore, skipped tests, deleted assertions,
  unimplemented stubs, empty error handling, or unexplained build exclusions.
- No generated, vendor, reference, or hardware-only exclusion without an
  explicit path, rationale, owner, expiry, and user approval below.
- Do not weaken this file to make a change pass.
- A host test or firmware build never substitutes for physical acceptance.

## Accepted numerical constraints

| Dimension | Rule | Checker | Stage |
| --- | --- | --- | --- |
| File size | At most 300 physical lines per non-generated `.go` file, including tests | Repository file-length guard | edit and task end |
| Function size | At most 60 lines | `golangci-lint` `funlen` | edit and task end |
| Complexity | Cyclomatic complexity at most 10 | `golangci-lint` `cyclop` | edit and task end |
| Changed coverage | At least 90% of changed executable lines | `covercheck` | task end and CI |
| Total coverage | Must not decrease from the accepted baseline | Go coverage profile ratchet | task end and CI |
| HTML input | At most 32 KiB encoded | Protocol and boundary tests | edit, task end, and target |
| Render latency | At most 5 seconds as a measured target | Pico stage timings | target acceptance |
| Render watchdog | 15 seconds only for stalled execution | Pico watchdog diagnostics | target acceptance |
| Local task-end gate | At most 90 seconds | Timed task-end wrapper | task end |

The 300-line limit is a readability and modularity bound. The 60-line and
complexity-10 limits keep embedded state transitions reviewable. Changed-line
coverage at 90% forces new behavior to carry evidence without pretending that
legacy coverage is already complete. The total-coverage ratchet prevents the
existing suite from becoming weaker.

Panel initialization, SPI transfer, physical refresh, and BUSY waiting are not
part of the five-second software-rendering target. They have independent stage
timings and diagnostics.

The 90-second local budget keeps the gate usable. A check that cannot reliably
fit moves to full CI or the physical-acceptance stage; it is not skipped. The
budget must not kill a valid check midway and interpret that interruption as a
product result. Full CI and physical acceptance have no 90-second limit.

## Enforcement

V2 feature code uses these reproducible checks:

1. `tools/go.mod` pins `golangci-lint` v2.12.2 and configures `funlen.lines: 60` plus
   `cyclop.max-complexity: 10`.
2. `covercheck` v0.2.0 is pinned and its parser/evaluator is reused by the local
   `worktree-covercheck`. The wrapper also measures staged, unstaged, and
   untracked code, includes `cmd/**`, and fails closed for executable statements
   in TinyGo-only files missing from host profiles. Upstream `covercheck` alone
   cannot do this because it reads only `base...HEAD` and excludes `cmd/**`.
3. `scripts/check-go-file-lengths.sh` enforces 300 physical lines and permits
   only the exact untouched legacy baseline. A modified legacy oversized file
   must first be split below 300 lines.
4. `quality/total-coverage-baseline.txt` ratchets the measured 75.0% total.
5. `scripts/quality.sh fast`, `task`, and `full` are the supported gates. `task`
   reports and then fails if its completed run exceeds 90 seconds; it never
   kills an in-progress check and misreports the interruption as a code result.
6. `scripts/quality.sh build` and `resources` expose the isolated TinyGo
   compile, size, stack, and allocation probes.

## Measured baseline and debt

Measured in the TinyGo Dev Container on 2026-09-02:

- `go test ./...` passes for `experiments/03-remote-epaper`.
- Overall statement coverage is 75.0%. This is the initial no-decrease baseline,
  not the target for changed code; changed executable lines still require 90%.
- Six existing Go files exceed 300 lines:
  - `protocol/receiver.go`: 306;
  - `protocol/codec.go`: 360;
  - `cmd/epaperctl/main.go`: 367;
  - `protocol/decoder_test.go`: 388;
  - `protocol/receiver_test.go`: 528;
  - `panel/driver_test.go`: 580.
- Full baseline lint reports 50 existing issues: 33 `cyclop`, 14 `errcheck`,
  two `staticcheck`, and one `unused`. Changed whole files are linted, so this
  debt cannot be copied into or retained in a modified file.

This is measured pre-existing debt, not an exception. These files must not grow.
A change that needs to modify one must first split it below the accepted limit
without changing behavior, with the existing tests kept green.

Sources:

- https://golangci-lint.run/docs/linters/configuration/
- https://github.com/golangci/golangci-lint/releases/tag/v2.12.2
- https://pkg.go.dev/github.com/dr-dobermann/covercheck@v0.2.0
- https://go.dev/doc/build-cover
- https://go.dev/doc/modules/managing-dependencies#tools

## Exceptions

Historical status, 2026-09-07: the following renderer-specific deferrals apply
to preserved checkpoint `37a1471`, not the active Go renderer. Their original
approval, failure and expiry evidence remains below. The exact Rust sources
were removed during E1 only after all three original geometry cases had active
passing replacements in `engine/historical_regression_test.go`. Additional
composition, text and alpha-normalization oracles were ported before deletion.
See `experiments/03-remote-epaper/docs/blitz-removal-map.md` for the exact manifest
and replacements. No ignore, renewed exception or lower quality threshold is
authorized for the native engine by these historical deferrals.

### BLITZ-IMG-001/002 — user-approved known-bug deferral, 2026-09-06

- Approval: after the earlier automated-review rejection, the user explicitly
  clarified: "ні, підтримка зображень має бути, але ці тести лише пропустити".
  This supersedes the earlier proposed condition that image delivery stay off.
- Exact path: `experiments/03-remote-epaper/blitz-probe/tests/images.rs`.
  Only `images_use_intrinsic_size_and_position` and
  `image_contain_and_clipping_use_final_layout` may use reasoned `#[ignore]`.
- Rationale: retain the two reproducible Blitz layout defects and their original
  expected pixels, but allow image integration and independent work to proceed
  with those known limitations. This is accepted debt, not a repair or full CSS
  conformance. Both reproductions still compile and remain explicitly runnable.
- Owner: pico-sandbox manager-renderer maintainer.
- Expiry: next change to the pinned renderer dependency set or image-layout
  implementation. Re-evaluate both cases then; remove ignores when fixed.
  Renewing the exception requires explicit user approval.
- Guards: real-worker successful image rendering and malformed BZR2 rejection
  must be tested. Alpha, resource limits, wire validation and all other tests
  remain active. No vendor patch, security relaxation or hardware change is
  authorized by this exception.
- Report: `experiments/03-remote-epaper/docs/blitz-image-layout-bugs.md`.

### BLITZ-POS-001 — user-approved engine-bug deferral, 2026-09-06

- Approval: "якщо це проблема ядра то не виправляй, пропускай задокументувавши
  і тести для цього вимкнувши". The reproduction and official Blitz status
  confirm an engine layout limitation, not an adapter or device fault.
- Exact path: `experiments/03-remote-epaper/blitz-probe/tests/viewport.rs`.
  Only `static_ancestor_does_not_capture_absolute_containing_block` may use a
  reasoned `#[ignore]`; keep its original expected pixels and test body intact.
- Rationale: continue independent implementation without maintaining a renderer
  patch/fork. This is accepted debt, not corrected positioning or complete CSS
  acceptance. The test still compiles and runs explicitly with `--ignored`.
- Owner: pico-sandbox manager-renderer maintainer.
- Expiry: next change to the pinned renderer dependency set or positioning
  implementation. Run the exact reproduction then; remove the ignore when
  fixed. Renewing this exception requires explicit user approval.
- Guards: all other viewport, composition, grammar, bounds, security and manager
  tests remain active, apart from the two separately approved image exceptions.
  No engine patch, dependency upgrade, runtime workaround, hardware change or
  blanket skip of later failures is authorized by this exception.
- Report and re-enable command:
  `experiments/03-remote-epaper/docs/blitz-positioning-gap.md`.
