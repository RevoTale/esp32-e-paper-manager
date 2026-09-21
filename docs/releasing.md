# Release boundaries

The publishing workflow is prepared, not yet exercised on GitHub. No image
digest exists from this migration. Do not treat example image names as available
releases.

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

Current base images use explicit version tags, not digest pins. Resolving and
recording base-image digests remains part of the independent image acceptance;
the current configuration does not promise byte-identical builds.

Sources: [Docker multi-platform Actions](https://docs.docker.com/build/ci/github-actions/multi-platform/),
[GitHub reusable workflows](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows).
