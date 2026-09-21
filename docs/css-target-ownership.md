# CSS targets and bitmap ownership

Historical selector/bitmap design, superseded 2026-09-07: this Rust component
was removed with the checked IPC worker. The native Go profile has one HTML5
tree and inline-only styles, not selector-based fallback authorization. See
[SPEC-engine.md](../SPEC-engine.md) and [the removal map](blitz-removal-map.md).

`blitz-probe::domscope::SceneScope` maps a native selector to actual elements in
one already parsed Blitz document. This is an isolated manager-side foundation,
not fallback rendering or a new public-input entry point. BZR3 still uses the
existing grammar-only path; Pico, USB, SPI and refresh behavior are unchanged.

## Contract

```rust
let mut scope = SceneScope::new(&document)?;
let targets = scope.select(".dashboard .value,#status")?;
```

- The scope borrows `BaseDocument` immutably. It walks actual DOM children,
  not source indentation, layout children, anonymous boxes or paint order.
- Each element is `Native` or `Bitmap(root_node_id)`. The outermost
  `data-epaper-render="bitmap"` owns itself and its descendants. A nested marker
  stays inside that same atomic subtree; it does not create an independent layer.
- The marker value must be exactly `bitmap`. Unknown/empty values fail even
  under an already marked ancestor. Non-HTML namespaces, namespaced attributes
  and template subtrees fail closed in this initial canonical-tree profile.
- Each selector is parsed once with the existing native admission gate, then
  matched by Blitz's `Node::matches_selector_raw`. Pinned Blitz reports
  `NoQuirks` even without a doctype: class/ID case stays significant. This API
  follows that renderer behavior; it does not claim browser quirks conformance.
  Results are unique per node and returned in DOM preorder. Duplicate HTML IDs
  can match multiple nodes; an ID string is not used as an ownership key.
- A comma list may return native targets and several different bitmap owners.
  There is no source-parent argument: a `<style>` inside a bitmap section still
  applies globally. HTML parser repair can also move a source-nested element
  outside a marker; the resulting tree decides its owner.
- No match means only that the current scene has no target. It grants no CSS
  waiver. Discard results across scene changes, even if IDs/classes look equal.

`Target` contains document-local Blitz `NodeId` metadata. It is not a stable
render-object ID, scene identity, authorization token or cross-revision cache
key. The borrow prevents ordinary DOM mutation while the scope exists; returned
metadata must not be used after mutation or against a different document.

## Bounded analysis

These are fixed provisional analysis caps, not new public API configuration.

| Limit | Accounting |
|---|---|
| 4096 nodes | All reachable DOM nodes, including document/text/comments |
| 64 depth | Edges from the document root at depth zero |
| 256 queries | Every `select` call, stylesheet rule and inline binding, including no-match calls |
| 16384 targets | Cumulative returned matches/inline targets; repeated queries count again |
| 32768 CSS bytes, 4096 tokens, 16 CSS nesting | Shared selector/style-source budget via `csscheck::Validator` |

A duplicate/cyclic child visit, broken parent link, missing/mismatched node or
invalid document root fails with `InvalidTree`. An anonymous layout box reached
through actual DOM children is rejected as `UnsupportedTree`. Validate before
matching: Blitz's `owner_doc` follows parents without a cycle guard. The node cap
bounds the visited set; the depth cap bounds recursive
traversal. Elements are indexed once per scope. Matching scans that bounded list
per query; no full-frame buffer, persistent matching cache or extra dependency
is introduced. Selector complexity still depends on the upstream matcher.
These caps do **not** establish a CPU deadline, peak-memory or energy measurement.

The inventory runs after DOM creation, so it does not protect HTML parsing,
resource fetching, style resolution or rendering before/after that point.
Future public integration still needs pre-parse limits and process isolation.
The `select()` method alone does not discover or validate whole-document CSS.
The new opt-in `styles()` method adds bounded source discovery, specified
declaration spans and rule/inline binding against this same tree, reusing the
shared validator/query/target budgets. See `css-style-bindings.md` for source,
rule and declaration limits and diagnostic precedence.

Any query error closes the scope. Return only the first source-free diagnostic;
never return a successful prefix of an over-budget query. The caller must discard
the entire analysis, including earlier results. A corrected scene uses a fresh
scope. `Css` contains a half-open UTF-8 byte span in the submitted selector, not
an HTML offset. Other errors contain no source text or resource URLs.

## Remaining integration gates

1. Local rule/inline binding is implemented in `styles()`. BZR4 now separately
   verifies Go/Blitz source identity before paint without activating ownership
   policy; see `css-source-identity.md`. Local DOM IDs are not Go source ordinals.
2. Enforce native/fallback property policy per target, including mixed and
   currently unmatched selectors, rather than granting a whole-rule waiver.
3. Check computed/inherited bounds and cross-boundary layout effects. Ownership
   alone says nothing about pixels painted outside the subtree.
4. Resolve complete-scene clipping, z-order, group opacity and overlapping
   backdrops before extracting bitmap pixels and assigning stable object IDs.
5. Keep asset/font/active-content limits and public-input isolation independent.

No bitmap marker repairs the three approved engine defects. See
`native-css-selectors.md`, `render-conflicts-and-module-boundaries.md` and ADR-009.

## Verification and sources

Real Blitz DOM fixtures cover mixed owners, nested markers, HTML repair, global
styles, escaped names, duplicate IDs, new-scene changes, exact resource limits,
malformed trees, source-free failures and case-sensitive matching with/without
doctype. No fixture is newly skipped.

Run inside the existing Dev Container from `blitz-probe`:

```sh
/root/.cargo/bin/cargo test --locked --offline --test css_target_ownership --test css_target_limits
```

Exact installed Blitz 0.3.0-beta.2 / Stylo 0.20.0 sources were inspected before
implementation; tests exercise these APIs, not an independent selector matcher.

- [Blitz selector matching](https://github.com/DioxusLabs/blitz/blob/67edf2061121382b3e43af19977fc3472aa7e069/packages/blitz-dom/src/query_selector.rs#L271)
- [Blitz DOM and layout children](https://github.com/DioxusLabs/blitz/blob/67edf2061121382b3e43af19977fc3472aa7e069/packages/blitz-dom/src/node/node.rs)
- [Blitz parent-chain traversal](https://github.com/DioxusLabs/blitz/blob/67edf2061121382b3e43af19977fc3472aa7e069/packages/blitz-dom/src/stylo.rs#L268)
- [Blitz fixed NoQuirks mode](https://github.com/DioxusLabs/blitz/blob/67edf2061121382b3e43af19977fc3472aa7e069/packages/blitz-dom/src/stylo.rs#L204)
- [Stylo author-origin selector parsing](https://github.com/servo/stylo/blob/67faaab3ff7aa66780ec1d0f51ca47e177b812d3/style/selector_parser.rs#L60)
- [WHATWG style-element semantics](https://html.spec.whatwg.org/multipage/semantics.html#the-style-element)
