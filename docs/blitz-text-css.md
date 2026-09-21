# Blitz text and cascade checkpoint

Historical checkpoint, superseded 2026-09-07: the Rust worker and old preview
were removed. Native text/pre-wrap/clipping preservation tests are recorded in
[the removal map](blitz-removal-map.md); current commands use
[engine-preview](native-preview.md). The measurements below still describe Blitz.

Measured 2026-09-06 against pinned Blitz 0.3.0-beta.2 in the existing Linux
arm64 Dev Container. These fixtures define verified behavior of our actual
`render_mono` path, not acceptance of the entire product HTML/CSS profile.
No production renderer, dependency, Pico firmware or refresh policy changed.

## Active pixel guards

`blitz-probe/tests/text.rs` uses the same Go Regular font as the manager.
One `OnceLock` lookup resolves the existing `golang.org/x/image` module via
readonly `go list`, with module networking disabled, then reads its bundled TTF.
Missing Go/module/font fails the test; nothing is skipped or downloaded. There
is no copied font, new dependency or reliance on installed fallback fonts.

| Fixture | Verified boundary |
| --- | --- |
| Normal whitespace | Repeated spaces/newline match the collapsed single-space rendering |
| Normal wrapping | Two words in a 90px box match an explicit line break |
| Nowrap | Same line as a wide box, with no second-line ink |
| Pre | Forced breaks survive; repeated spaces independently match two nonbreaking spaces |
| Pre-wrap | Soft wrapping plus an explicit newline produces three reference lines |
| Hidden overflow | Exact mask of visible glyphs at 45×12, 90×28 and 90×40, including mid-glyph clipping |
| Inheritance | Nested text inherits size/color and matches explicit styling |
| Overflow-wrap | `anywhere` and `break-word` split a fixed-width long word; normal does not |
| Text-align | Real glyph bounds move left/center/right; integer translation preserves their pixels |

Reference-render comparisons include non-empty ink guards. Clipping uses an
independently masked visible frame, not another clipping implementation.
These tests do not establish every glyph shape, fallback font, language,
intrinsic/min-content width or whitespace boundary case.

`tests/cascade.rs` adds four full-bitmap guards: specificity (including compound
and descendant selectors), equal-specificity source order, important versus
normal inline precedence, and invalid-declaration recovery. Observing a parser
feature does not add it to the product allowlist. In particular, the raw worker
silently discards invalid CSS; it cannot replace typed profile diagnostics.

## Known gaps and decisions

- **Ellipsis:** the local preview's identical overflowing strings look the same
  under clip and `text-overflow:ellipsis`, without an ellipsis. No handling was
  found in the locked DOM/paint source. This remains an unimplemented product
  capability; it is not an approved approximation or a new ignored test.
  The preview is the current reproducer; there is no automated ellipsis
  acceptance oracle yet. Recheck before admitting the value to the profile.
- **Pre-line / break-spaces:** the pinned Stylo-to-Parley mapping explicitly
  maps both unfinished modes to `Preserve`. Do not claim correct semantics from
  successful parsing. No vendor patch or new ignore was added.
- **Viewport root:** the first preview omitted definite root/body height and
  lost its bottom-anchored footer. Setting `html,body` to 100% width/height
  restored it. Keep that explicit fixture precondition; auto-height root/body
  behavior is not accepted by this check.
- **Product gate:** bounded CSS grammar/value validation with typed diagnostics,
  unsupported-value policy, missing glyph diagnostics, general content overflow
  and timestamp layout occupancy remain open. Do not expose this trusted-local
  worker directly to arbitrary public HTML based on these tests.

The two approved image-layout ignores and enabled image path are unchanged.
This increment makes no measured CPU, RAM, bandwidth or energy improvement claim.

## Reproduce

Inside the already-running Dev Container, from the experiment directory:

```sh
/root/.cargo/bin/cargo test --locked --offline --manifest-path blitz-probe/Cargo.toml --test text --test cascade -j2
go run ./cmd/blitz-preview ./blitz-probe/target/debug/epaper-blitz-probe ./testdata/blitz-text.html ./blitz-probe/target/preview-text.png
```

The real Go → BZR1 → Blitz PNG has readable Ukrainian including Ґ/Є/І/Ї, six
text cards and a local-preview footer. It is not a screenshot from the panel.
No USB traffic, firmware flash or device acknowledgement occurred.

Final Rust verification: 39 tests pass, zero fail, exactly the two existing
approved image-layout ignores. All-target Clippy with warnings denied and
format check pass. Independent review's repeated-space correction and optional
directional alignment guard are both included; no required finding remains.
The wider Go task gate also passes in 16s: zero lint issues, changed-worktree
coverage 93.5%, total 90.1%, USB and Wi-Fi TinyGo builds successful. Six existing
file-length debts remain reported. These numbers do not measure Rust coverage
or physical display acceptance.

## Sources

- [CSS whitespace and wrapping](https://www.w3.org/TR/css-text-3/#white-space-property).
- [Emergency word wrapping](https://www.w3.org/TR/css-text-3/#overflow-wrap-property).
- [Clipping and ellipsis semantics](https://www.w3.org/TR/css-overflow-3/#text-overflow).
- [Cascade ordering](https://www.w3.org/TR/css-cascade-5/#cascade-sorting).
- [CSS parser error recovery](https://www.w3.org/TR/css-syntax-3/#error-handling).
- [Pinned Blitz whitespace mapping and TODO](https://github.com/DioxusLabs/blitz/blob/67edf2061121382b3e43af19977fc3472aa7e069/packages/blitz-dom/src/stylo_to_parley.rs#L281).
- [Pinned Blitz wrap mapping](https://github.com/DioxusLabs/blitz/blob/67edf2061121382b3e43af19977fc3472aa7e069/packages/blitz-dom/src/stylo_to_parley.rs#L358).
