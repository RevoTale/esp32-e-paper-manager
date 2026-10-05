# Runtime dependency notice audit

Scope: Linux `epaper-manager`, CGO disabled, Go 1.26.8, current `go.mod` on
2026-09-21. This is a source inventory, not legal clearance for distribution.
The repository MIT license covers our code, not every dependency or font.

Maintainer decision, 2026-10-03: keep the pinned scanx dependency and proceed
without further upstream license investigation. The scanx finding below is
informational, not a PR or publication blocker. Do not open an upstream issue
or replace the dependency for this finding. This decision does not establish
new license terms or change the remaining notice/source packaging work.

## Evidence

In the matching Dev Container, enumerate compiled packages rather than the
entire module graph (which also includes unused GUI backends and test modules):

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go list -deps \
  -f '{{with .Module}}{{if .Version}}{{.Path}} {{.Version}}{{end}}{{end}}' \
  ./cmd/epaper-manager | sort -u
go mod verify
```

The amd64 and arm64 module/version lists matched exactly. There were 27 external
modules; module-cache verification passed. This is a compilation dependency
inventory, not proof that the linker retains every function or asset.

## Findings requiring release review

| Dependency | Version | Evidence and unresolved work |
| --- | --- | --- |
| `github.com/benoitkugler/textprocessing` | `v0.0.6` | Root LICENSE contains LGPL-2.1. `engine → canvas/text → textprocessing/fribidi` is an actual package import. Review the applicable package terms and static-binary distribution requirements; do not relabel it MIT. |
| `github.com/srwiley/scanx` | `e94503791388` | Informational; maintainer accepted proceeding without further investigation. No root LICENSE in the downloaded module. `scan.go` declares FreeType License OR GPL-2.0-or-later and refers to a missing LICENSE file; `span.go` has no initial license header. |
| `github.com/BurntSushi/freetype-go` | `b763ddbfe298` | Root LICENSE offers FreeType License or GPL. Record the selected permitted route and preserve its complete notices. |
| `github.com/golang/freetype` | `e2365dfdc4a0` | Root LICENSE likewise offers FreeType License or GPL. Treat separately from the fork above. |
| `github.com/go-fonts/latin-modern` | `v0.3.3` | Both LICENSE and LICENSE-GUST exist; embedded fonts need their own notice treatment, not only the Go wrapper license. |
| `modernc.org/knuth` | `v0.6.0` | LICENSE, KNUTH-LICENSE and LICENSE-STAR-TEX exist. Preserve all applicable notices, not just the first matching filename. |

Other observed root notices: fpdf, brotli, textlayout, canvas, font, minify,
parse and goldmark include permissive license texts; graphics-go, xgb,
poly2tri-go, rasterx, serial, x/image, x/net, x/sys, x/text, token and star-tex
also carry root license texts. xgbutil includes COPYING. typesetting declares
Unlicense OR BSD in its root LICENSE. These observations do not replace a
file-level audit or collection of the full notice text.

## Distribution boundaries

The current Dockerfile copies only the application binary; it does not yet
package a complete application dependency notice bundle. Debian package
copyright files are a separate base-image concern. Go standard-library/runtime
notices, embedded font/data notices and the ESP-IDF firmware dependency closure
also require explicit coverage. Historical TinyGo notices under `docs/licenses`
do not satisfy these current artifacts' inventory.

Attribution follow-up:

1. Review the applicable LGPL/dual-license distribution path without assuming
   that publishing our source alone satisfies every condition.
2. Collect exact notices and required source/build materials for the released
   artifacts, preserving dependency versions and integrity evidence.
3. Include the notice bundle in both runtime architectures and inspect it in
   the built images. Repeat the inventory when dependencies change.

Do not remove a dependency, change renderer behavior, select a new project
license, or declare legal compliance merely to silence this audit.

## Pinned upstream references

- [textprocessing license](https://github.com/benoitkugler/textprocessing/blob/v0.0.6/LICENSE)
- [scanx revision](https://github.com/srwiley/scanx/tree/e94503791388)
- [Latin Modern fonts](https://github.com/go-fonts/latin-modern/tree/v0.3.3)

The findings above came from local module contents validated by `go mod verify`.
Initial web retrieval of the first two URLs failed. Follow-up GitHub API
inspection succeeded: the pinned scanx root and its current default-branch root
both lack a LICENSE/COPYING file. This is not just a truncated Go module cache.
The pinned textprocessing `fribidi` directory contains its own LICENSE; that
local package-specific text also identifies LGPL version 2.1. Thus the LGPL
finding is not based solely on an unrelated module-root license.

The scanx issue inventory contained one open issue, unrelated to licensing:
[prevent infinite loop, #1](https://github.com/srwiley/scanx/issues/1). Its report
is not proof that our renderer reaches the reported condition; do not claim it
as a reproduced product defect. No upstream issue/comment was created, no
license text was invented, and no dependency was replaced during this audit.

## Recheck — 2026-10-03

Re-enumerated both Linux architectures from the current candidate: the same
27 module/version pairs; `go mod verify` passed. Local inventories are ignored
`build/runtime-modules-amd64.txt` and `build/runtime-modules-arm64.txt`.
GitHub retrieval of the pinned scanx `scan.go` confirmed its FreeType-or-GPL
header; the module still contains no LICENSE file and `span.go` starts directly
with its package declaration. This does not establish permission for all
adaptations by that project's authors. No replacement license was fabricated.

The subsequent maintainer decision above removes the scanx clarification
requirement. Remaining work is the notice/source/rebuild bundle for the selected
distribution terms, including LGPL and ESP-IDF components. No upstream contact,
dependency replacement or change to GitHub publication settings was made.
