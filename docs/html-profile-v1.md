# Bounded dashboard HTML profile v1

> This is the older direct on-device/USB compatibility profile. The manager
> profile, including `<img>` and explicit server raster fallback, is specified in
> `manager-html-css-profile-v2.md` and does not add image decoding to Pico.

This is a static dashboard format, not a browser. USB and network transports
must deliver one complete validated update envelope before parsing starts.
Transport fragmentation therefore cannot change HTML semantics.

## Limits

| Resource | Maximum |
|---|---:|
| encoded UTF-8 input | 32,768 bytes |
| content nodes | 256 |
| element depth | 16 |
| attributes per element | 8 |
| CSS declarations | 512 |
| raw text | 24,576 bytes |
| input assets | 0 bytes |
| tokenizer forward work | 131,072 bytes |

The parser is iterative. It borrows the input and caller-provided node storage;
node text, names, and attributes are views into the immutable input.
Whitespace-only formatting nodes are discarded.

## Syntax

The optional HTML doctype is accepted case-insensitively. Comments are ignored.
Tags and attribute names are ASCII case-insensitive. Elements must be properly
nested; implicit closing and error recovery are deliberately unsupported.

Supported elements:

```text
html head meta title body main header footer section article div
h1 h2 h3 h4 p span strong em ul ol li br hr
```

`meta`, `br`, and `hr` are void. Other elements require an explicit closing
tag unless written with `/>`.

Supported attributes:

```text
id class style data-priority lang charset aria-label
```

Values must be single- or double-quoted. Event handlers, links, scripts,
inputs, external resources, SVG, and arbitrary `data-*` attributes are
rejected. The CSS phase separately validates `style` contents.

Supported named text entities are `amp`, `lt`, `gt`, `quot`, `apos`, and
`nbsp`. Numeric and other named entities are rejected. Decoding occurs during
text measurement/rasterization so parsing does not create a second text copy.

## Failure behavior

Malformed markup, mismatched closing tags, unsupported tags/attributes/entities,
and every limit breach return typed errors before framebuffer mutation. No DOM
is returned on error. Arbitrary valid UTF-8 input is fuzzed to ensure parsing
cannot panic or escape the caller-provided node bound.
