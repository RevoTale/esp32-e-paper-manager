# Development

Open this checkout in VS Code's Dev Container. When already inside the parent
project's approved container, work from this submodule directory. Do not start a
second development container automatically.

For noninteractive Docker commands use the image's normal environment or
`bash -c`, not `bash -lc`: the base image's login profile replaces PATH and can
hide Go. Source `$IDF_PATH/export.sh` when invoking SDK tools; `make firmware`
already does this.

```sh
make quality
```

The gate covers three Go modules: the root application, `tools`, and
`firmware/esp32/tests/interop`. `make interop` builds native C peers before
exercising the Go/C boundary; root `go test ./...` cannot cover that module.

Focused checks: `make format lint`, `make test coverage`, `make native`,
`make tinygo`, `make firmware`, `make build`. Firmware build instructions printed
by ESP-IDF are not authorization to flash a connected board.

For review against a branch, set `QUALITY_BASE_REF` to its Git revision. Do not
lower coverage or change the baseline to conceal a regression. Read
[the quality contract](CONSTRAINTS.md) before changing tests or gates.

Production readiness requires independent image builds, immutable published
digests, isolated API validation and visible hardware acceptance. A green local
gate alone is insufficient. The migration map is `docs/migration-manifest.tsv`;
historical source context is under `docs/history`.
