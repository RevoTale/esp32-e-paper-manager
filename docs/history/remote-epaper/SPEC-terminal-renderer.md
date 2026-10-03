# Spec: terminal-renderer

Status: approved automatically by user policy on 2026-08-30.

## Objective

Render bounded local text or piped command output into the panel frame on the
host, using the self-contained official Go `basicfont.Face7x13` bitmap font.

## Contract

- Input is at most 1 MiB and produces a fixed 114-column by 36-row snapshot.
- Newlines, carriage returns, backspace, four-column tabs, wrapping, and
  bottom scrolling are deterministic.
- Printable ASCII is rendered directly; other runes use the replacement glyph.
- CSI and OSC escape sequences are discarded, never interpreted or executed.
  Other control bytes are ignored.
- Rendering is black on white and emits the same exact 48,000-byte frame as
  image conversion.
- CLI mode is explicit: `epaperctl -text FILE` or `epaperctl -text -` for stdin.

## Verification

Unit tests cover escape stripping, controls, wrapping/scrolling, input bounds,
and frame equivalence. CLI tests cover file and stdin loading.

Source: https://pkg.go.dev/golang.org/x/image/font/basicfont
