# Native-Go renderer dependency qualification

## Firmware boundary regression, 2026-09-07

Adding atomic authoring edits exposed an existing package-boundary problem:
`devicelink/manager.go` shared a package with the device client, importing the
whole manager on TinyGo. The new DOM parser's initialization then increased
the legacy Wi-Fi control build to 831,672 flash / 179,876 static RAM bytes.
Marking that server-only adapter `!tinygo` reduced the exact same build to
691,164 / 107,932. No GPIO, panel or Wi-Fi authentication behavior changed.

The task/build gate now checks `tinygo list -deps -target=pico2-w -tags=wifi
./cmd/device` for forbidden manager/new-engine/Canvas/parser imports before
building. A Go build-constraint test guards the adapter itself. The unified
firmware must extend this gate to its own target and omit legacy MCU HTML too.
This is static image-size evidence, not heap-peak or electrical power evidence.

Native Linux preview and Darwin arm64 preview/USB/manager binaries also built
with `CGO_ENABLED=0`. Final artifacts must be rebuilt after remaining migration
changes; the dependency/security/notice audit below is still a release gate.

## Initial dependency qualification

2026-09-06. Applies only to the manager's selected runtime closure, not every
optional exporter, GUI, GPU or command in upstream repositories.

| Component | Pinned module | Role / license evidence |
| --- | --- | --- |
| HTML5 | golang.org/x/net v0.58.0 | canonical DOM; BSD-style LICENSE |
| CSS / raw attribute lexer | github.com/tdewolff/parse/v2 v2.8.16 | lexical/grammar validation; MIT LICENSE |
| Canvas | github.com/tdewolff/canvas dae8cd8e19a71648a574a6e3d7ffcda38a961635 | native CPU raster/text; MIT LICENSE |
| Text shaping | github.com/go-text/typesetting v0.3.4 | native Go shaping; Unlicense OR BSD-3-Clause |
| Bidi | github.com/benoitkugler/textprocessing v0.0.6, fribidi package | LGPL-2.1; preserve license/source/rebuild route for distribution |
| Embedded font | golang.org/x/image/font/gofont | fixed Go font bytes; retain font license |

This table records directly inspected boundaries, not a claim that all transitives
have been exhaustively audited. Before release, inventory the actual imports
and their notices; do not label the entire graph permissive-only. Do not enable
Canvas's optional latex/harfbuzz/fribidi tags, external executables or network fonts.

Verified in the existing Dev Container:

- `CGO_ENABLED=0 go test ./engine`: PASS, exact pixel viewport, white backdrop,
  top-left coordinates, Cyrillic text draws ink with embedded font.
- `CGO_ENABLED=0 go list -deps -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' ./engine`:
  no CgoFiles in selected runtime closure.
- `GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ./engine`: PASS.

These qualify the adapter, not the unfinished HTML/layout engine. Full manager
cross-build, dependency advisory audit and binary distribution notices remain
final release gates. Adding Canvas also upgrades transitive x/exp; TinyGo's
actual selected closure must still be rebuilt and measured.

Primary references: [Canvas](https://github.com/tdewolff/canvas),
[CSS parser API](https://pkg.go.dev/github.com/tdewolff/parse/v2/css),
[HTML5 API](https://pkg.go.dev/golang.org/x/net/html),
[Fribidi package license](https://github.com/benoitkugler/textprocessing/blob/v0.0.6/fribidi/LICENSE).

## Patched native toolchain, 2026-09-07

The first symbol-level `govulncheck v1.7.0` audit of manager, preview, USB and
provisioning commands found eight standard-library advisories under Go 1.26.2.
These included TLS post-handshake input handling and `os.Root` path traversal;
the latter is relevant to enrollment file access. This is dependency evidence,
not a claim that every reported path is exploitable in this application.

Main and tools modules now select `toolchain go1.26.8`, the patched release in
the same minor line, through Go's automatic toolchain mechanism **inside the
existing container**. Both `go env GOROOT` and `tinygo env GOROOT` resolve to
that downloaded module toolchain. No host installation or container rebuild
was performed. TinyGo 0.41.1 compiles the unified candidate against it.

Repeated audit: **No vulnerabilities found**, across the four selected commands,
28 modules and Go 1.26.8. Preserve both local reports:
`build/engine-host-vulnerability-audit.txt` (before) and
`build/engine-host-vulnerability-audit-patched.txt` (after). This result is dated;
repeat the pinned scanner against the current vulnerability database for each
release. It does not replace application review or radio-firmware qualification.

Primary sources: [Go release history](https://go.dev/doc/devel/release#go1.26),
[Go toolchains](https://go.dev/doc/toolchain),
[TLS advisory](https://pkg.go.dev/vuln/GO-2026-6090),
[os.Root advisory](https://pkg.go.dev/vuln/GO-2026-4970),
[Go security practices](https://go.dev/doc/security/best-practices).

## Native renderer sample measurement

2026-09-07, Linux/arm64 Dev Container, Go 1.26.8, 800×480 dashboard, five
iterations each: warm renderer **24.8 ms/op, 12,033,657 B/op, 30,305 allocs/op**;
cold renderer including fonts **30.4 ms/op, 18,763,028 B/op, 53,461 allocs/op**.
These are cumulative allocated bytes per render, not peak retained memory or
energy measurements. The sample is not a worst-case latency guarantee. HTML
and these allocations stay on the manager, never on Pico. Reusing the renderer
avoids repeated font construction; identical confirmed output suppresses device
transfer, not the necessary author-scene render used to establish equivalence.

Bounded fuzz run: PackBits 3,985,202 executions/15 seconds; HTML renderer
106,009 executions/~16 seconds, both PASS. Fuzzing used two workers; it is bug
discovery evidence, not a proof that all inputs terminate or render correctly.
Reproduce with `go test ./engine -run '^$' -bench BenchmarkDashboard
-benchtime=5x -benchmem`, and the package `FuzzRenderer`/`FuzzCodec` targets.
