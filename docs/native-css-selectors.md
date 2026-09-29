# Native selectors and bitmap-fallback boundaries

Historical Rust/selector scope, superseded 2026-09-07: this component and its
worker were removed. Current native rendering deliberately accepts inline-only
CSS and has no selector-driven bitmap fallback. Follow
[SPEC-engine.md](history/remote-epaper/SPEC-engine.md) and [the removal map](blitz-removal-map.md).

`csscheck::Validator::native()` checks the selector syntax admitted by the
manager profile as well as declaration values. It remains opt-in: BZR3 still
uses grammar-only validation. This increment does not enable bitmap fallback,
implement a second selector matcher, or change Pico firmware.

## Admission contract

Allowed: tag names, classes, IDs, compounds, descendant combinators and comma
lists. Examples: `div.card#status`, `.dashboard .value`, `html,body`.
Escaped punctuation remains part of an identifier: `.card\:active` is a literal
class name, not `:active`. Unicode identifiers follow the locked parser.

Outside the native profile: universal selectors, explicit namespaces, child
and sibling combinators, attribute selectors, all pseudo-classes/elements and
nesting. They are not stripped or approximated. Some already fail the pinned
standalone grammar (`:has`, `:nth-child(... of ...)`, `::first-line`); grammar
errors remain `InvalidSelector` rather than being relabeled as capability errors.

Validation order for each selector prelude:

1. Existing shared document byte/token/depth limits.
2. Stylo selector grammar and the strict visitor rejecting recovered branches.
3. Original cssparser tokens: identifiers, ID hashes, dot and comma only, with
   ordinary whitespace/comments handled by cssparser.
4. The rule's declaration checks, including empty or overridden rules.

The token gate is not a grammar: Stylo has already validated combinations and
structure. It preserves a distinction an optimized AST loses: selectors 0.40.0
intentionally removes redundant `*|` when no default namespace exists. An
AST-only allowlist incorrectly admitted `*|div`; the negative test remains a
regression guard. No substring tests or custom tokenizer are used.

`UnsupportedSelector = 14` reports the original whole prelude as a half-open
UTF-8 byte span in decoded CSS. It includes trailing selector whitespace if
present, but not the declaration block. Existing codes are unchanged; no source
text enters diagnostics. Code 14 is local to opt-in validation and is not emitted
over BZR3. Future activation must update both IPC diagnostic implementations.

## Fallback is not stylesheet scope

`data-epaper-render="bitmap"` changes the intended paint representation, not
CSS matching. A `<style>` inside that subtree still applies document-wide.
The real-render test places `#outside{width:10px}` inside a marked section:
an external box changes from four to ten pixels. This is expected HTML behavior,
not an engine defect and not evidence that fallback is implemented.

Keep these integration gates explicit:

- Determine ownership from actual matched targets in the tree being rendered,
  never only from the `<style>` element's parent or source ordinal. Parser-repaired
  trees and implied elements must preserve checked identity across Go and Blitz.
- A selector list can target native and bitmap nodes at once. A property outside
  native capability cannot become authorized for the native targets merely
  because another match is inside a bitmap subtree.
- Re-evaluate ownership for each submitted scene: class/ID changes, movement,
  insertion and removal can change rule targets. A currently unmatched rule is
  not a permanent safety guarantee.
- Inheritance and layout can cross the paint boundary. Validate resolved sizes,
  clipping and resource work even when all direct matches lie inside fallback.
- Resolve z-order, group opacity and overlapping backdrops in the canonical
  complete scene before extracting pixels. A cropped subtree is not independent
  merely because it has a marker; see `render-conflicts-and-module-boundaries.md`.
- Active content, URLs, fonts and resource limits retain their separate gates.
  A marker never bypasses them. Same-engine fallback does not repair the three
  documented Blitz layout defects.

Actual DOM ownership/matching is now available as a separate opt-in foundation:
see `css-target-ownership.md`. Rule/declaration integration, subtree extraction
and stable render-object identity are still pending. The current grammar worker
must not be presented as a public-input sandbox.

## Verification and cost

Eight added tests cover admitted and rejected selector families, namespace
normalization, escaped literals, UTF-8 truncations, source-free diagnostics,
document budgets, unchanged BZR3, real cascade pixels and the unscoped-style
boundary. Existing tests and approved engine reproductions remain intact.

Run in the existing Dev Container from `blitz-probe`:

```sh
/root/.cargo/bin/cargo test --locked --offline --test css_native_selectors --test css_native_selector_render --test checked_wire
```

Native mode adds a bounded token walk per selector prelude. No retained DOM,
matching cache, new dependency, wire data or Pico allocation is introduced.
CPU/peak-memory/energy effects have not been benchmarked; passing fixture timings
are not an energy measurement.

## Sources

- [W3C selector structure and terminology](https://www.w3.org/TR/selectors-4/#structure)
- [W3C specificity](https://www.w3.org/TR/selectors-4/#specificity-rules)
- [WHATWG style element](https://html.spec.whatwg.org/multipage/semantics.html#the-style-element)
- [Locked namespace normalization](https://github.com/servo/stylo/blob/67faaab3ff7aa66780ec1d0f51ca47e177b812d3/selectors/parser.rs#L2929)
- [Locked selector visitor](https://github.com/servo/stylo/blob/67faaab3ff7aa66780ec1d0f51ca47e177b812d3/selectors/visitor.rs)

The exact installed selectors 0.40.0 / Stylo 0.20.0 source was inspected before
implementation; the specification is a semantics reference, not proof of engine
coverage. Declaration/value policy: `native-css-declarations.md`.
