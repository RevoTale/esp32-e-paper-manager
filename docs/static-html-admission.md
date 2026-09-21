# Static HTML admission

Historical implementation, superseded 2026-09-07: the Go/Blitz adapter was
removed. Current admission uses one bounded native HTML5 tree and inline-only
CSS under [SPEC-engine.md](../SPEC-engine.md). The original boundary and tests
below are checkpoint evidence; see [native replacements](blitz-removal-map.md).

Implemented 2026-09-06 in Go `blitzworker`, enforcing ADR-009 on the canonical
tree before image decoding, serialization and worker startup. Bitmap markers
do not bypass these rules. This is one admission gate, not a full sanitizer,
complete HTML/CSS profile or OS sandbox. Trusted-author/loopback limits remain.

## Rejected behavior

- Script elements of every type, including data scripts.
- Forms and input/button/select/textarea controls, their grouping/options.
- Embedded documents/plugins, frames, applets and parameter elements.
- Audio/video/source/track elements.
- Interactive details/summary/dialog containers, including already-open ones.
- Every parsed attribute name beginning `on`, even unknown or empty handlers.
- Navigation attributes `href`, `ping`, `target`, `download`, `action`,
  `formaction`; embedded-document `srcdoc`.
- `autofocus`, `contenteditable`, customized built-in `is`, `popover`,
  `popovertarget` and `popovertargetaction`.
- All `meta http-equiv` pragmas, not only refresh redirects.

Use static containers/text for interactive widgets and `<a>` without navigation
attributes for a plain label. Unsupported behavior is rejected, not silently
removed. Ordinary metadata/charset, titles, comments, escaped code examples,
data-* attributes and scripting-disabled noscript text remain permitted by this
gate. Other image/CSS/profile validation can still reject their contents.

Inspection follows x/net's parsed tree and normalized names. A string resembling
`<script>` inside a comment or text is not an executable element. Discarded HTML
tokens are not checked as if they survived canonical serialization. The existing
32-KiB input, 4096-node and depth-64 limits still apply. No second traversal,
image buffer or Pico RAM is added; energy savings have not been measured.

## Errors and evidence

Return only static `ErrInput`; `Worker.Render` adds the existing source-free
`InvalidDocument`/`HTML` diagnostic before spawning a renderer. No partial HTML,
images, CSS or frame is returned. Manager rejection keeps the confirmed scene
and waits for corrected input, using its existing recovery contract.

Five tests cover native/bitmap paths, uppercase/empty attributes, static
counterexamples, rejection before PNG decoding and rejection before a nonexistent
worker can launch. Initial tests demonstrated that active nodes passed. Review
then found details/summary, reproduced separately and fixed without engine edits.
Locked Blitz does implement details toggling; this was an admission-policy gap,
not demonstrated active execution in the one-shot renderer.

Final task gate PASS 17s, 93.6% changed / 90.2% total Go coverage; the new gate
has 23/23 changed executable lines covered. Worker/manager/diagnostic race tests
pass. No extra ignore or dependency. See `../VERIFICATION.md`.

## Remaining boundary

This denylist does not authorize every other tag/attribute, CSS URL, font,
background asset, computed size or fallback property. Those complete-profile
and resource-isolation gates remain open. Raw Rust BZR1/2/3/4 callers do not
inherit this Go-only admission guarantee. Never expose raw IPC publicly or
remove trusted-author/loopback restrictions on the strength of this check.

Sources inspected:

- [WHATWG script/event mechanisms](https://html.spec.whatwg.org/multipage/webappapis.html#event-handler-content-attributes)
- [WHATWG meta pragma directives](https://html.spec.whatwg.org/multipage/semantics.html#attr-meta-http-equiv)
- [Pinned Blitz details activation](https://github.com/DioxusLabs/blitz/blob/67edf2061121382b3e43af19977fc3472aa7e069/packages/blitz-dom/src/events/pointer.rs#L695)
- [x/net canonical reserialization](https://pkg.go.dev/golang.org/x/net/html#hdr-Security_Considerations)
