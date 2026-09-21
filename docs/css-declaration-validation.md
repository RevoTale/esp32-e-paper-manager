# Strict CSS declaration validation

Historical worker validation, superseded 2026-09-07: the Rust grammar adapter
was removed. Current inline-only validation lives in `engine/style` under
[SPEC-engine.md](../SPEC-engine.md). The commands/status below refer to
checkpoint `37a1471`; see [the removal map](blitz-removal-map.md).

Status: grammar validation is integrated into the manager's checked local worker
path. It is **not** the native capability whitelist, complete HTML sanitizer or
public-input sandbox. Do not substitute it for those remaining gates.

An isolated opt-in native declaration core now layers canonical property,
expanded-longhand and specified-value checks on this parser. Default/BZR3 remain
grammar-only. See [core policy and remaining gates](native-css-declarations.md).

## Contract

Stylesheet integration increment: `validate_stylesheet(&str)` accepts ordinary
qualified style rules, reuses Stylo selector grammar and shares the declaration
budget across the whole sheet. At-rules (including otherwise skipped initial
`@charset`) and nested style rules are rejected. All error spans refer to the
whole effective stylesheet. The same byte/token/depth limits apply. This is
still grammar validation, not native selector/property capability acceptance.

`csscheck::validate_declarations(&str)` validates one decoded CSS declaration
list (an inline `style` value or the body of a style rule). Empty lists are
valid. It does not accept a whole stylesheet, parse selectors, fetch URLs,
resolve variables, calculate layout, paint or change the display.

- Reuse the exact cssparser 0.37.0 and Stylo 0.20.0 already locked by Blitz.
  Stylo owns property/value grammar, shorthand expansion and `!important`.
  Do not recreate CSS tokenization or vendor-patch the renderer.
- Check every declaration, including an invalid value followed by a valid value
  for the same property. Browser-style recovery is not successful validation.
- Reject unmatched delimiters, unfinished escapes and unterminated strings,
  URLs, comments and blocks, including parser recovery at end of input.
- Bound a document's CSS to 32 KiB, 4096 tokens (including whitespace/comments/closers), 16
  nested blocks/functions and 256 declarations. Reject non-finite numeric
  tokens before semantic parsing. These are component resource bounds, not
  property-specific limits. `Validator` shares bytes/tokens/declarations across
  all inline and stylesheet sources; discard it on the first error.
- Return one typed error with a half-open UTF-8 byte range in this **decoded
  CSS input**, never a location in the original HTML. Do not include source
  text or upstream error strings in logs. Structural/resource errors take
  precedence over semantic errors; within each pass, fail at the first error.

Known-but-out-of-profile CSS can pass this grammar layer. A following native
capability gate must still reject unsupported properties/values, fonts, asset
references and computed limits. In particular, parsing `var(...)`, ellipsis,
Grid or a URL does not prove resolution, painting or authorization. The raw
Blitz capability fixtures and image-enabled preview remain available unchanged.
No USB/Pico wire format or firmware changes are part of this component.

## Why an extra boundary is needed

Blitz beta.2 `DocumentConfig` does not expose an error reporter. Stylo's normal
reporter also intentionally suppresses some invalid duplicate declarations and
unknown vendor-prefixed properties. Call a fresh
`DeclarationParserState::parse_value` for each declaration instead of relying
on a successful render or reporter silence. A separate bounded token walk
detects syntax recovery before Stylo parses values.

Primary references, checked against the locked local sources:

- [cssparser declaration callback](https://docs.rs/cssparser/0.37.0/cssparser/trait.DeclarationParser.html)
- [cssparser parser and nested-block API](https://docs.rs/cssparser/0.37.0/cssparser/struct.Parser.html)
- [Locked Stylo declaration parser](https://github.com/servo/stylo/blob/67faaab3ff7aa66780ec1d0f51ca47e177b812d3/style/properties/declaration_block.rs#L1538)
- [Locked Stylo EOF escape recovery](https://github.com/servo/stylo/blob/67faaab3ff7aa66780ec1d0f51ca47e177b812d3/style/custom_properties.rs#L1305)
- [CSS Syntax error recovery](https://www.w3.org/TR/css-syntax-3/#error-handling)

## Verification / integration

The original thirteen active tests cover valid shorthand/importance/escaped names, invalid
duplicates, unknown properties, malformed/truncated syntax, exact resource
boundaries, nested functions, non-ASCII byte ranges and source-free diagnostics.
Independent review caught EOF identifier repair and uncounted nested closers;
both now have RED→GREEN regressions, including valid escape controls.

Original declaration-only checkpoint: 52 Rust tests pass, exactly two approved image ignores;
fmt/all-target Clippy pass. Go task gate passes in 18 seconds with unchanged
firmware sizes. Review has no remaining required finding. No hardware update.

From `blitz-probe` inside the existing Dev Container:

```sh
/root/.cargo/bin/cargo test --locked --offline --test csscheck --test csscheck_limits
```

`MAX_*` limits now aggregate across a document. Token scanning and Stylo parsing are separate
bounded passes; this trades extra manager CPU for rejection before semantic
work. There is no added Pico RAM, SPI or network activity. Manager allocation,
latency and energy are not yet benchmarked. The built-in parser may scan the
rest of a block while unwinding an error, still within the byte input limit.

Integration adds six stylesheet/selector tests, two document-budget tests and six
checked-wire/process tests. Stylo's forgiving invalid `:is`/`:where` branches and
out-of-context `&` are rejected. Pinned Stylo disables `:has` and
`:nth-child(... of ...)`; active negative tests record those unavailable features.
Ordinary selectors and nested `:is`/`:where`/`:not` controls stay active.

Go discovers sources in its canonical HTML tree. BZR3 validation precedes Blitz
DOM construction; real subprocess tests prove source-free BZE1 rejection before
font processing and continued image output. The manager exposes revision-bound
diagnostics and waits for corrected input, preserving the confirmed scene.
See [checked IPC and recovery](blitz-checked-ipc.md) for the exact trust boundary.

Still required: native property/value capability map, fallback policy, complete
HTML security/resource isolation and remaining renderer capability gaps. Passing
these grammar checks does not accept arbitrary public HTML.
