# CSS source and declaration bindings

Historical cross-parser design, superseded 2026-09-07: Rust `SceneScope` and
BZR bindings were removed. One native HTML5 tree now owns inline declarations
under [SPEC-engine.md](history/remote-epaper/SPEC-engine.md). Earlier commands and measurements
remain checkpoint evidence; see [the removal map](blitz-removal-map.md).

`SceneScope::styles()` discovers CSS in the actual Blitz DOM and binds specified
declarations to native/bitmap targets. This is opt-in manager-side metadata, not
fallback authorization, a cascade implementation or a new public renderer API.
BZR4 now checks Go/Blitz source identity but keeps grammar-only policy;
legacy BZR3/default validation is unchanged. Firmware and S5 are unchanged.

## Data contract

```rust
let mut scope = SceneScope::new(&document)?;
let bindings = scope.styles()?;
```

The scope borrows the same `BaseDocument` used for selector matching. Use it
before layout creates anonymous boxes; discard bindings after DOM mutation or
across scenes. Node IDs are document-local, not stable render-object IDs.

| Field | Meaning |
|---|---|
| `source` | Actual DOM `NodeId` plus `Stylesheet` or `Inline` |
| `selector_span` | Original stylesheet prelude byte range, including trailing whitespace; `None` for inline styles |
| `declarations` | Original property/value byte ranges in this decoded CSS source |
| `targets` | One shared ordered target list per rule or inline source |

Ranges are half-open UTF-8 byte offsets, not HTML positions. Property ranges
preserve escapes; value ranges retain whitespace/comments and `!important`.
An inline range addresses the HTML-decoded attribute value. A stylesheet range
addresses concatenated direct text children. No raw CSS, ID strings, values or
URLs are retained in output or diagnostics.

Keep every specified declaration, including overridden duplicates. Stylo still
owns property grammar, shorthand expansion and cascade during rendering; this
metadata does not select winning declarations or computed values. Storing
targets once per rule avoids a declaration-by-target cross-product.

## Discovery, validation and ownership

Sources follow DOM preorder: a `<style>` stylesheet first, then its own `style`
attribute, if any. Rules and declarations keep their source order. This order
is deterministic discovery, **not** a replacement for CSS cascade precedence.
Inline styles bind directly to their element without inventing an ID selector.
Stylesheet selectors are parsed once and reused by Blitz's actual matcher.

Ownership follows `css-target-ownership.md`. A sheet inside a bitmap subtree
can target outside elements. Mixed owners and unmatched rules are retained;
all declarations still pass the native core. A marker or zero matches grants
no property waiver. Unsupported `background` shorthand, for example, stays
rejected even inside a bitmap marker; supported `background-color` is separate.

Discovery rejects `base`/`link` elements and unsupported stylesheet context:
`disabled`, `scoped`, nonempty types other than trimmed case-insensitive
`text/css`, or nonempty media other than trimmed case-insensitive `all`.
Non-text direct stylesheet children are rejected. These checks mirror Go's
canonical discovery. BZR4 now verifies their source identity separately; see
`css-source-identity.md`. Local bindings alone do not provide that guarantee.
Asset, font and active-content gates remain independent.

Pinned Blitz additionally HTML-decodes stylesheet RAWTEXT. Reject any `&` in
that raw CSS before binding so checked and executed text cannot differ through
this decode. Inline attributes are already HTML-decoded and need no second
decode. This is a compatibility guard, not an engine patch.

All sources are collected before CSS grammar validation; discovery errors can
therefore precede a syntax error in an earlier source. Structural CSS checks
precede semantic validation. A failure returns only `StyleError { source,
cause }`, no successful prefix, and closes the scope. Discard earlier analysis
results too; corrected input needs a fresh scope. `source: None` denotes a
scope/tree-level failure without an attributable CSS source.

## Bounds and cost

| Limit | Accounting |
|---|---|
| 256 style sources | Includes empty inline/style sources; checked before copying the next source |
| 32768 copied CSS bytes | Cumulative discovered source payload; checked before cloning/appending text |
| 32768 parsed bytes / 4096 tokens / 16 CSS nesting | Shared validator budget across `select()` and every style source |
| 256 declarations | Shared across all stylesheets/inline sources in this scope |
| 256 rules per collected stylesheet | Includes empty/unmatched rules; typed `RuleLimit` before collecting rule 257 |
| 256 queries | Cumulative `select()` calls, stylesheet rules and inline bindings |
| 16384 targets | Cumulative matches per rule/query plus one per inline binding, not per declaration |

DOM limits remain 4096 nodes / depth 64. Empty sheets consume source slots but
no query; empty inline attributes consume both. Calling `styles()` again repeats
analysis against remaining shared parser/query/target budgets, not a cache.
The source payload cap bounds copied text, not total allocator capacity or
peak process RAM. Metadata and matching work have separate bounds; no measured
latency/energy improvement is claimed.

Default validation passes no metadata collector, so it allocates no new
declaration/rule vectors. `RuleLimit` code 15 belongs only to opt-in collection;
it is not emitted by BZR3. No dependency or wire-format change is introduced.
The pre-parse 32-KiB encoded HTML gate and future process isolation remain
necessary: this API accepts an already constructed local DOM.

## Regression checks and remaining integration

Run inside the existing Dev Container from `blitz-probe`:

```sh
/root/.cargo/bin/cargo test --locked --offline --test css_style_bindings --test css_style_limits --test css_style_paint
```

Fourteen active tests cover mixed owners, global sheets, inline entity decoding,
exact escaped/Unicode spans, overwritten declarations, source-local private
diagnostics, native policy without waivers, exact shared budgets and pre-copy
source-slot rejection. The paint test analyzes and then resolves the **same**
document, compares it with normal rendering, and asserts exact cascade pixels.
It is a locked-renderer regression test, not a physical panel test.

Checked Go/BZR4 source identity is implemented without enabling target policy
in submissions. Full property/asset policy, computed/inherited bounds,
cross-subtree layout effects, clipping/z-order/alpha extraction and stable render
objects remain open. This increment does not repair or add to the three approved
Blitz engine ignores. See ADR-009 and `render-conflicts-and-module-boundaries.md`.

## Pinned sources

Exact installed upstream files were inspected; docs.rs web retrieval failed
during this increment, so local versioned API documentation was the evidence.

- [cssparser 0.37.0 rule/declaration callbacks](https://docs.rs/cssparser/0.37.0/src/cssparser/rules_and_declarations.rs.html)
- [Stylo specified declaration parsing](https://github.com/servo/stylo/blob/67faaab3ff7aa66780ec1d0f51ca47e177b812d3/style/properties/declaration_block.rs#L1538)
- [Blitz stylesheet RAWTEXT decode](https://github.com/DioxusLabs/blitz/blob/67edf2061121382b3e43af19977fc3472aa7e069/packages/blitz-dom/src/document.rs#L1135)
- [Blitz actual selector matching](https://github.com/DioxusLabs/blitz/blob/67edf2061121382b3e43af19977fc3472aa7e069/packages/blitz-dom/src/query_selector.rs#L271)
