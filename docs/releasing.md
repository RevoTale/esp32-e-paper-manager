# Release boundaries

The publishing workflow is prepared, not yet exercised on GitHub. No image
digest exists from this migration. Do not treat example image names as available
releases.

Publication also requires resolution of the current
[runtime dependency notice audit](runtime-dependency-notices.md). In particular,
the renderer imports an LGPL-labelled package. The scanx finding is informational
and no longer blocks PR or publication, per the maintainer's 2026-10-03 decision.
No blanket MIT-only dependency claim has been established,
and the runtime image does not yet include a complete application notice bundle.

An explicitly authorized version-tag push triggers the complete reusable quality
workflow, then publication to `ghcr.io/revotale/esp32-e-paper-manager`. Publication
is restricted to this repository, version tags and commits reachable from main.
Only the publishing job has package-write permission; it uses GitHub's scoped
token, not a stored personal credential. Protect main and release tags through
repository rules before allowing unattended releases.

The image contains Linux amd64/arm64 manifests, revision metadata, SBOM and
provenance. The workflow summary records its immutable digest. Deployment should
pin that digest, not a mutable tag. There is deliberately no automatic `latest`
promotion or deployment/restart of an existing manager.

Before calling a release production-ready, verify:

1. All quality jobs actually completed for the release commit.
2. Both architectures can start with isolated test credentials.
3. API authentication and readonly/non-root operation work in the image.
4. The published manifest and revision match the selected commit.
5. Authorized hardware delivery is visible and recovery/rollback was checked.
6. `make latency` passed separately in the recorded qualification environment;
   ordinary `make quality` does not certify real-time scheduling latency.

Current base images use explicit version tags, not digest pins. Resolving and
recording base-image digests remains part of the independent image acceptance;
the current configuration does not promise byte-identical builds.

Sources: [Docker multi-platform Actions](https://docs.docker.com/build/ci/github-actions/multi-platform/),
[GitHub reusable workflows](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows).

## Release automation — prepared 2026-10-03

Publication is fail-closed: release-please and both publishing jobs require
repository variable `RELEASE_PUBLICATION_APPROVED=true`. Leave it unset until
the remaining notice/source packaging work is resolved and the maintainer explicitly approves
publication. Quality still runs while publishing jobs are gated. A green run
with skipped publication is not a release. This guard also applies to manual
version tags; it does not grant permission to push one.

`release-please.yml` runs Quality before proposing/updating a release PR on
main. Conventional commits drive the version and changelog; the first version
is0.1.0. Review/merge is manual. No auto-merge or device flashing occurs.

Create repository secret `RELEASE_PLEASE_TOKEN`: a repository-scoped credential
with Contents, Pull requests and Issues read/write. Prefer a dedicated bot;
set an expiry and arrange rotation. Never paste the token into logs or chat.
The normal `GITHUB_TOKEN` intentionally has no fallback: events it creates do
not trigger downstream release-PR/tag workflows. Repository/organization policy
must permit this automation. Keep Quality required in main branch protection.

After the release PR merge, release-please creates the prerelease/tag. The tag
starts `publish.yml`: Quality validates that exact revision and uploads the
single firmware archive; separate jobs attach that same-run artifact and publish
the existing amd64/arm64 GHCR manager image. Both require Quality success and a
main-branch ancestor. Release assets contain checksums and source COMMIT.

GitHub release creation and asset upload are not atomic: a prerelease can exist
while checks/builds run. Do not install until Publish release succeeds and all
assets exist. Failed publication is not a successful release. Asset overwrite
is forbidden; inspect partial publication before retrying and prefer a new
version. No automatic rollback, release deletion or force-push is performed.

Public download is an application-only update for already installed boards;
see [firmware bundle instructions](firmware-release.md). First-install bootloader
and partition files remain in the documented source-build flow. There is no
second experimental firmware asset. All release archives select region support;
full-only builds remain regression checks, not distributed alternatives.

## Local verification — 2026-10-03

`make workflows` passed actionlint and packaging tests: archive allowlist,
checksums, exact application bytes, full-only rejection, oversize rejection and
no overwrite. Inverting the capability guard made the test fail; restoring it
passed. `make firmware-partial` built the full/partial application successfully;
the real archive passed checksum verification in `build/release-check.e6zdyB`.
This local archive is from an uncommitted working tree, not a published release.

Full `make quality` stopped at lint in the pre-existing ignored manual runner
`build/items-soak/main.go`: cyclop at `run` and `await`, and unchecked body Close.
No exclusion or suppression was added. Later gates did not execute in this run;
do not report this as a passing full gate. GitHub execution, token setup and
publication remain unverified. No commit, push, release or hardware action ran.

## Activation boundary

### Candidate recheck — 2026-10-03

A clean candidate copy in the existing Dev Container passed the entire
`make quality` (exit 0). The copy contains tracked files plus non-ignored
untracked changes over baseline `1b60448ddf2fd594bf3e05d3ceafc593f28412c3`;
it is not that committed revision and is not a published release.
No lint rule, test, coverage floor or package exclusion was changed.

- Root coverage: 94.8% on all three uncached runs with identical sorted
  covered-block profiles. Changed executable coverage: 95.8%.
- All three Go lint scopes, race/vet/module verification, workflow/package
  tests and vulnerability audit passed (no vulnerabilities reported).
- Native functional tests: 24/24; C function-size, Go/C interoperability,
  TinyGo checks, ESP32 full-only/partial-capable builds and Linux amd64/arm64
  manager builds passed.
- Separate latency test passed: read 69,116 us, write 54,652 us, cancellation
  27,125 us. These are host measurements, not ESP32 timing acceptance.
- The real partial-capable application packaged successfully; outer checksums
  passed. Archive has exactly firmware.bin, README.md, LICENSE, COMMIT and
  SHA256SUMS. This validation archive uses the baseline as a local label, not
  an immutable identity for the uncommitted candidate; never distribute it.

Local evidence: `build/quality-clean-candidate.log` and
`build/latency-clean-candidate.log`. They remain ignored.
The original checkout still contains the ignored one-off soak runner, which
`go list ./...` discovers despite Git ignoring it. Its lint failure is not
fixed by the clean-copy result. No soak source/assertion was deleted or excluded
from an existing gate to make the result green.

Removed six obsolete Pi5/Pico/ESP32 task-list copies; immutable recovery links
are in [the documentation index](README.md). Specifications, ADRs, failure
records and license evidence remain. Subsequent edits in this recheck concern
documentation only.

GitHub execution still requires explicit commit/push authorization. No remote
workflow, release, image publication, token setting or hardware test ran.
The remaining notice/source packaging work, excluding the waived scanx
investigation, remains a publication blocker; green Quality is
not license clearance or continuous hardware endurance evidence.

Local workflow/package checks are not GitHub execution. Setup is inactive until
changes are reviewed/pushed/merged, the secret is installed, and repository
permissions are confirmed. Before broad distribution review ESP-IDF/component
redistribution notices; prototype acceptance is not a legal or production audit.

## Sources

- [Release Please action and token behavior](https://github.com/googleapis/release-please-action)
- [Manifest configuration](https://github.com/googleapis/release-please/blob/main/docs/manifest-releaser.md)
- [Configuration schema](https://github.com/googleapis/release-please/blob/main/schemas/config.json)
- [ESP-IDF flashing workflow](https://docs.espressif.com/projects/esp-idf/en/v5.5.5/esp32/api-guides/tools/idf-py.html)
