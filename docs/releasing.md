# Automatic releases

Each push to main runs Quality, then pinned go-semantic-release calculates the
version from Conventional Commits and creates the tag and GitHub release.
The same workflow calls publish.yml to attach firmware and publish the manager.

No approval variable, personal token, release PR or manual tag push is required.
The standard GITHUB_TOKEN has Contents write for release creation and Packages
write for the image. Organization policy can restrict those declared permissions.

feat commits advance minor versions, fix commits advance patches. Breaking
changes advance major versions after 1.0; before 1.0 they advance the minor.
Initial development starts at 0.1.0. Documentation-only commits need no release.
Workflow dispatch can retry publication for a version already on that commit.

One tag identifies all products:

- Go client module: github.com/RevoTale/esp32-e-paper-manager@vX.Y.Z.
  See [client usage](go-client.md).
- Firmware: epaper-esp32-7in5-v2.tar.gz, SHA256SUMS and COMMIT on the release.
- Manager image: ghcr.io/revotale/esp32-e-paper-manager:vX.Y.Z, Linux amd64/arm64.
  Deploy by the recorded digest when reproducibility matters.

Publishing downloads Quality's artifact from this same workflow run, checks its
checksum and COMMIT, and verifies the tag resolves to the checked source.
One full/partial-capable ESP32 application is distributed; full-only builds are
regression/recovery checks. See [firmware instructions](firmware-release.md).

GITHUB_TOKEN-created tags do not trigger another push workflow. Publishing is
an explicit reusable-workflow call after versioning, with no dependence on that
suppressed event. Main releases are serialized without cancelling publication.
Quality remains enabled for pull requests and branch pushes.

Release creation and firmware/image publication are separate operations. Use
the release after its Release workflow succeeds and all assets and the image
exist. Diagnose failed jobs before retrying; assets are never silently
overwritten. CI does not flash devices.

## Verification

Quality covers lint, race/vet/module checks, three stable coverage runs, reachable
vulnerability audit, native C and Go/C interoperability, TinyGo checks, ESP32
applications and Linux builds. make latency is a separate host qualification.
Build/protocol success does not establish panel lifetime or continuous hardware
endurance. Hardware evidence is in [the candidate record](partial-candidate-20261001.md).

[Dependency attribution](runtime-dependency-notices.md) remains informational;
it introduces no publication toggle. The project MIT license does not relabel
third-party dependencies.

## Sources

- [go-semantic-release action](https://github.com/go-semantic-release/action)
- [Versioning](https://github.com/go-semantic-release/semantic-release)
- [GitHub token events](https://docs.github.com/en/actions/how-tos/writing-workflows/choosing-when-your-workflow-runs/triggering-a-workflow)
- [Go module publication](https://go.dev/doc/modules/publishing)
