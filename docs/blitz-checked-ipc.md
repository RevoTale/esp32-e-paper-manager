# Checked renderer IPC

This is trusted local Go-manager → Rust-worker IPC, not the Pico protocol or a
public endpoint. One worker process handles one update. BZR1/BZR2 remain raw
capability-probe inputs; they do not promise strict public HTML validation.

## Current BZR4 request

Go always emits BZR4, including an explicit empty source manifest. It keeps the
BZR3 header below but permits source count zero; each source appends an element
ordinal u32 after its CSS length, before the CSS bytes. The worker verifies
source kind/order/ordinal/text against the actual DOM before image injection or
paint. A mismatch is static `CSS_IDENTITY`, nonzero exit, no frame; Go treats it
as fatal ErrWorker, not a normal client CSS rejection. No downgrade retry.
See [identity contract and tests](css-source-identity.md).

## Legacy BZR3 request

All integers are little-endian. The initial 16-byte header keeps BZR1's width,
height, font length and canonical HTML length, with magic `BZR3`. It is followed
by image count (u32, 0–16) and CSS-source count (u32, 1–256), then font and HTML.
Each CSS source contains kind (u8: 1 inline, 2 stylesheet), three zero reserved
bytes, UTF-8 byte length (u32), and CSS. Existing BZR2 image records follow.
Trailing bytes are invalid. Existing viewport, font, HTML and image limits apply;
all CSS sources share 32 KiB, 4096 tokens, 16 nesting levels and 256 declarations.

Go creates the source manifest from the same bounded HTML tree it serializes,
before image decoding. It retains each source's canonical document-order element
ordinal (including implied html/head/body) for diagnostics. It does not send a
second DOM or start a second worker. This legacy path trusts the manifest's association
with HTML; a malicious local IPC caller can lie, so this interface must not be
exposed as an untrusted network service.

The worker validates all sources before constructing the Blitz DOM. This is a
grammar gate, not yet the complete native CSS-value capability/security gate.
Go rejects template/link/base and incompatible style contexts. `<style>` text
containing `&` is rejected because pinned Blitz decodes entities again there;
inline attributes use normal HTML decoding. Invalid CSS is never silently
repaired; the HTML parser still performs its documented tree normalization.

## Replies

Success remains BZM1. A handled CSS rejection returns exactly 16 bytes and exit
status zero: `BZE1`, code (u16), zero-based source index (u16), start and exclusive
end UTF-8 byte offsets (u32 each) in the decoded CSS source. No raw input, URLs or
dependency error text are sent. Codes: 1 input limit, 2 token limit, 3 nesting
limit, 4 declaration limit, 5 non-finite number, 6 syntax, 7 unknown property,
8 invalid value, 9 invalid selector, 10 unsupported rule. These IDs are stable.

Go verifies exact reply size, code, source index, range and UTF-8 boundaries
before constructing a typed diagnostic. Unknown/malformed replies are worker
output failures. IO/runtime failures remain nonzero exit status with static
stderr categories; Go never forwards dependency stderr. No rejection is a frame
or an instruction to clear the display.

## Manager status and recovery

`PUT /v2/screen` remains asynchronous (202 means queued, not valid or displayed).
The authenticated `GET /v2/screen/status` adds optional `failure` with `revision`
and `diagnostic`: `code`, `source`, `element`, `start`, `end`. Sources are `inline`,
`stylesheet`, or `html`. Adapter-local HTML rejection uses code 11 with zero
location when parsing/budget/asset validation cannot provide a specific element;
11 is not a BZE1 code. No document/property/value text is reflected by status.

Check `failure.revision` against `current`. A new submission clears the old
failure; a stale render cannot attach its diagnostic to a newer revision.
`confirmed` is preserved when a document is rejected, because nothing was sent.
The pump consumes that revision once and waits for an explicit corrected submit.
There is no clear, retry, renderer restart or busy polling on a rejected scene.
Runtime/transport/queue/frame-contract failures remain fatal, including failures
coinciding with a newer scene. Transport ambiguity still requires resync.

Questions answered by this status: which submitted revision failed; where its
CSS failed; which revision remains protocol-confirmed. Physical visibility is
still a separate acceptance boundary, not inferred from `confirmed`.

Sources: [cssparser](https://docs.rs/cssparser/0.37.0/cssparser/),
[Blitz style processing](https://github.com/DioxusLabs/blitz/blob/67edf2061121382b3e43af19977fc3472aa7e069/packages/blitz-dom/src/document.rs#L1135),
[Stylo selector capabilities](https://github.com/servo/stylo/blob/67faaab3ff7aa66780ec1d0f51ca47e177b812d3/style/servo/selector_parser.rs#L586-L598).
