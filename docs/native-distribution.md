# Candidate source and notices

This is an internal, unflashed candidate, not a claim of completed public
distribution or hardware acceptance. Keep source and license files together
with binaries. Do not describe the dependency graph as permissive-only.

The native release script records exact module versions and copies license,
copying and notice files from selected host/firmware module trees, preserving
subpackage paths. The Go font license comes from `font/gofont/ttfs/README`.
`scanx` has license notices in its source files rather than a root license;
those files are retained alongside the FreeType dependency license. The CYW
module includes a separately licensed radio firmware binary; this is not Go
application source and must not be stripped from the notices set. Host notices
cover the Linux/arm64 and Darwin/arm64 CGO-disabled dependency union. Go's
license and hash-checked, pinned TinyGo/compiler-rt/picolibc/newlib license
texts are included separately. Audit tools are not distributed in the package.

The selected bidi dependency contains LGPL-2.1 subpackages. Candidate materials
include the complete pinned `textprocessing` and `textlayout` module sources,
plus the repository-visible Go source/module manifests and two embedded
synthetic HTML fixtures. No ignored enrollment, passwords, certificates or personal captures
are copied. Historical build artifacts and the original Rust checkpoint remain
in the project, outside this candidate archive.

The source list is NUL-delimited and rejects symlinks, ignored tracked sources
and nonregular inputs. The release snapshots source before building, then
checks the exact file list and hashes again before completing the package.

To rebuild a host command, unpack `project-go.tar.gz`, use the pinned Go
toolchain in `go.mod`, download verified dependencies with `go mod download`,
then run `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -buildvcs=false
./cmd/epaper-manager` (or the other command; use `GOOS=darwin` for macOS).
For a modified bidi library, unpack its source and use a local Go
module replacement before rebuilding. This route requires no proprietary
renderer worker. The source archive is a build/relink input, not a complete
repository/test-fixture mirror; use the repository for the full test suite.

Firmware requires TinyGo 0.41.1 and its RP2350 runtime/toolchain support. Use
`tinygo build -target=pico2-w -scheduler=tasks ./cmd/screen-device`. Generic
firmware contains no credentials; enrollment happens only over physical USB.
Before distributing publicly, review all included third-party notices and
the chosen source/rebuild distribution route; this engineering inventory is
not a legal certification.
