# Native preview and synthetic cycle stamp

2026-09-07. Maintainer preview of the native engine's pixels without USB,
network delivery or device confirmation. Run commands only in the existing
Dev Container from `experiments/03-remote-epaper`.

```sh
go run ./cmd/engine-preview -format rgba ./engine/testdata/dashboard.html ./build/native-rgba.png
go run ./cmd/engine-preview -cycle-start 2026-09-06T12:34:00Z -timezone Europe/Kiev ./engine/testdata/dashboard.html ./build/native-stamped.png
go run ./cmd/engine-preview -format frame -cycle-start 2026-09-06T12:34:00Z ./engine/testdata/dashboard.html ./build/native-stamped.bzm
```

The output parent directory must exist and the output file must not exist.
The tool creates mode-0600 files exclusively; it does not overwrite artifacts.
Input remains bounded to 32 KiB and uses the same inline-only HTML/CSS engine
profile as the manager. There is no Rust worker, asset URL fetch or host-font
lookup. Width/height default to 800×480; `-width` and `-height` select the logical
viewport before layout. The default output remains an unstamped monochrome PNG.

## Stamp contract

A nonempty `-cycle-start` explicitly enables the synthetic stamp. Its value
must be an RFC3339 instant with an offset; no current-time fallback is used.
`-timezone` defaults to `Europe/Kiev`, matching manager USB mode. Go timezone
data is embedded in this host command, not in the engine or Pico firmware.
Equivalent RFC3339 offsets identifying the same instant produce identical
pixels. At the example instant the default zone paints `2026-09-06 15:34`.

Stamped output supports `mono` and `frame` only. `rgba` remains the unquantized
engine preview and rejects `-cycle-start`; the shared stamp paints final 1-bit
pixels. `frame` preserves the existing bounded BZM1 file format.

`refreshstamp.Tracker.Begin` creates the label from the supplied instant. The
pixel path is `engine.RenderReserved` → quantized frame → `refreshstamp.Paint`.
It shares the manager's protected bottom-right rectangle and bitmap painter. The tracker
never receives `Complete`: writing a preview does not confirm a physical cycle.
The label is a synthetic operator input, not observed panel timing.

The native reservation clears its exact rectangle and emits a source-free
overlap warning when composed content occupies it. This deliberately follows
current manager semantics; the historical Blitz preview instead rejected an
occupied monochrome corner. The viewport is not reduced. Invalid stamp values,
too-small geometry, rejected documents and diagnostic-output failures return
before creating an output file. Disk-write failures can leave a partial new
file; an existing file is never replaced.

## Verification

```sh
go test -race ./cmd/engine-preview ./refreshstamp
go test -race ./engine -run 'TestHistorical|TestMigration'
```

Tests compare every stamped PNG/frame pixel with the shared stamp applied to
an independently constructed scene, retain unstamped format/default checks,
exercise equivalent instants/default timezone and verify failure-before-output.
The [removal map](blitz-removal-map.md) records preserved legacy assertions.
These are software checks, not visible panel, radio or power-loss acceptance.
