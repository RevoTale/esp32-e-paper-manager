# BLITZ-POS-001 — deferred engine positioning defect

Historical failure, superseded 2026-09-07: the Rust renderer was removed after
the exact static-wrapper geometry passed in active native Go tests. This is a
replacement, not an upstream Blitz fix. The original ignored test is preserved
at `37a1471`; see [the removal map](blitz-removal-map.md). Commands and status
below describe that historical checkpoint, not a current skipped test.

2026-09-06, pinned Blitz 0.3.0-beta.2, existing Debian arm64 Dev Container.
No dependency, renderer implementation or Pico firmware was changed.

Status: user-approved deferral. Only the exact reproduction below is ignored in
the default suite; its assertions remain unchanged. Positioning is not repaired
or fully accepted. Independent implementation can continue without a Blitz fork.

## Reproduction

The test `static_ancestor_does_not_capture_absolute_containing_block` in
`blitz-probe/tests/viewport.rs` renders a 24×16 viewport:

- An absolute outer box starts at (2,2), size 20×12.
- Its static child has size 8×6, with no border, padding, margin or transform.
- That wrapper contains a black absolute 2×2 child with `right:0;bottom:0`.

**Expected by CSS:** (20,12), the positioned outer box's bottom-right corner.
**Measured:** (8,6), the static wrapper's bottom-right corner. Exact packed pixels
fail; the child is painted, but relative to the wrong containing block. There is
no font, image, network, USB, clipping or timing dependency in this fixture.

```sh
# Inside the already-running Dev Container, from blitz-probe:
cargo test --locked --offline --test viewport static_ancestor_does_not_capture_absolute_containing_block -- --ignored --exact
```

The preceding ten viewport fixtures remain valid for their narrower cases;
none exercised an intervening static ancestor. The expected geometry must not be
changed to match the buggy output. The command above intentionally reproduces
the failure; a green default suite does not establish this capability. This has
its own root exception, separate from the two image exceptions.

## Evidence and impact

- [CSS 2.2 containing blocks](https://www.w3.org/TR/CSS22/visudet.html#containing-block-details):
  an absolutely positioned element uses its nearest positioned ancestor here.
- [Official Blitz status](https://blitz.is/status/css), checked 2026-09-06,
  explicitly documents missing static positioning and immediate-parent absolute
  positioning. The local fixture independently verifies the pinned build.
- [Related issue #690](https://github.com/DioxusLabs/blitz/issues/690) describes
  initial-containing-block/fixed issues and is closed in the retrieved page.
  It is related evidence, not this nested-wrapper reproduction or proof of a fix.

This violates required positioning semantics for ordinary nested dashboard
markup. It cannot be solved by removing timeouts, changing Pico pins, sending a
larger frame, grouping updates differently or marking the same subtree for
bitmap rendering: the same Blitz layout produced the wrong pixels already.
Reparenting HTML casually would also change cascade, inheritance, clipping and
stacking contexts; it is not an acceptable silent workaround.

The previous text gap remains independent: official status marks `text-overflow`
unsupported and links [Parley #304](https://github.com/linebender/parley/issues/304).
Pinned whitespace mapping also has explicit pre-line/break-spaces TODOs. See
`blitz-text-css.md`. Fixing this positioning test alone does not close those gaps.

## Approved decision and re-enable condition

On 2026-09-06 the user requested that confirmed engine bugs be documented and
their reproductions skipped, without fixing the engine. This supersedes the
earlier stop awaiting renderer-remediation scope. Preserve Blitz, the original
fixture and this known limitation; do not silently rewrite markup or expected
pixels. The exception does not mark the native/fallback profile complete or
waive separate text, input-isolation or device acceptance gates.

Owner: pico-sandbox manager-renderer maintainer. On the next pinned renderer
dependency or positioning implementation change, run the exact command above.
If it passes, remove the ignore and run the full suite. If it still fails,
renewal needs user approval. See `../../../CONSTRAINTS.md`, BLITZ-POS-001, for
the exact scope; no other new test skips are included.

Verification after deferral: default full Rust suite has 66 passes, exactly
three approved ignores (two image, one positioning), zero failures. Formatting
and all-target Clippy pass. Explicit `--ignored --exact` still reproduces the
same geometry failure. Go task gate passes in 17 seconds; no runtime code changed.

Unchanged verified work: checked BZR3 grammar/diagnostics and manager recovery
pass their focused tests; Go task gate passes, and real Go → Blitz image preview
still works. Latest physical checkpoint stays S5. This is a server renderer
limitation, not evidence of a broken Pico or e-paper screen.
