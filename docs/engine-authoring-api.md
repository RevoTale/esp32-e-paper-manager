# Native engine authoring API

Migration contract, 2026-09-07. The manager owns HTML, layout, pixels and batching;
the Pico receives prepared pixels, never these JSON edits or HTML trees.

`GET /v2/screen` returns current HTML and a strong per-process-epoch ETag.
`PUT /v2/screen` replaces HTML. `PATCH /v2/screen` accepts `application/json`:

```json
{"edits":[{"id":"status","text":"Ready"},{"id":"card","attributes":{"style":"background:white;color:black","title":null}}]}
```

All mutations require the same Bearer authentication and exact current
`If-Match`. Read the new ETag after an accepted mutation. HTTPS is mandatory for
external access; this document does not claim public-mode acceptance is complete.

## Atomic edit contract

- Body <=32 KiB, 1–64 edits, exact lowercase field names. Duplicate JSON keys,
  malformed Unicode, trailing values and unknown fields reject. Only attribute
  values accept null. `remove:false` and empty operations reject.
- `id` targets one existing unique ID. Each target appears once; targeting both
  an ancestor and descendant in a batch rejects regardless of operation order.
- `text` replaces children with literal text. `html` replaces children with a
  target-context HTML fragment. They are mutually exclusive; either may accompany
  `attributes`. `remove:true` is exclusive of other mutation fields.
- Attributes are exact tag-allowlisted names; `id` cannot change. Null removes
  an attribute. Root html/head/body attribute changes are allowed, but their
  removal/child replacement is not. Void elements cannot receive children.
- Fragments reject active/unsupported tokens and structural html/head/body/
  doctype tokens. Serialization and reparsing must preserve the intended
  canonical structure; HTML5 repairs that reparent nodes reject.
- The private final tree and all embedded assets are validated before revision
  CAS. A concurrent writer, bad final document or bad asset leaves markup,
  revision and batching queue unchanged. Layout/paint diagnostics are asynchronous,
  as for PUT. A renderer without document validation cannot expose PATCH (501).

The existing optional `debounce_interval` / `max_wait` queue groups accepted
complete scenes. It never combines independently painted rectangles or bypasses
device refresh safety. A batch of five related edits is one scene revision.

Responses: 202 queued; 400 invalid document/edit; 401 unauthorized; 412 stale
base; 413 body limit; 415 unsupported content type/encoding; 428 missing base;
429 rate limit; 501 renderer cannot validate edits. No source/asset bytes are
included in rejection bodies. Parsing/rendering does not fetch external resources.

## Pixel evidence

`GET /v2/screen/status` separates `delivered` (last actual terminal-success
revision) from `confirmed` (newest scene whose pixels are known equivalent).
Identical final pixels can advance `confirmed` without USB/network/SPI traffic.
Unknown delivery invalidates reuse; explicit device reset/session changes must
call `InvalidatePixels` when idle, or resolve an active delivery as unknown.
The manager copies only its confirmed baseline; this adds no Pico framebuffer.
Scheduled full-maintenance bypass and live session invalidation wiring remain
unfinished migration tasks and are not implied by unit-test acceptance.

## Primary references

- [RFC 9110 If-Match](https://www.rfc-editor.org/rfc/rfc9110.html#name-if-match):
  lost-update prevention; process epoch fences stale versions after restart.
- [Go JSON decoding](https://pkg.go.dev/encoding/json#Unmarshal): duplicate-key,
  Unicode replacement and case-insensitive-field pitfalls avoided here.
- [Go HTML fragment parsing](https://pkg.go.dev/golang.org/x/net/html#ParseFragment)
  and [Render caveat](https://pkg.go.dev/golang.org/x/net/html#Render): correct
  context does not alone guarantee a constructed tree survives reparsing.

## HTTPS and local secret boundaries (2026-09-07)

Fact: the native server requires TLS 1.3 and admits at most 16 simultaneous
HTTPS connections, including unfinished TLS handshakes. HTTP/2 permits eight
streams per connection, a 64 KiB connection receive window, a 32 KiB stream
receive window and 16 KiB incoming frames. Existing header/read/write/idle
timeouts remain active. These limits use Go's native HTTPS server and the
existing `x/net/netutil.LimitListener`; no custom TLS or HTTP runtime is used.

Fact: Bearer authentication precedes scene/status access and document parsing.
Exactly one Authorization field is required. Mutations require exactly one
Content-Type field and reject every Content-Encoding field, including empty or
repeated fields. Duplicate-field rejection removes interpretation ambiguity;
the earlier first-field behavior was not demonstrated to bypass token checking.
Both PUT and PATCH share two concurrent body/validation slots and the existing
30-attempts-per-minute budget. A full slot budget returns 429 immediately.
Body size remains at most 32 KiB, including chunked/unknown-length requests.

Fact: the authenticated author controls the document, not its I/O authority.
The native engine accepts its bounded HTML/inline-CSS profile and inline PNG or
JPEG assets. Documents cannot fetch URLs, read host files/fonts, execute scripts
or start processes. Rendering is serialized and carries node, asset, geometry
and paint-work limits. Error/status diagnostics contain numerical source
locations and codes, never submitted source or secret values. Authorized GET
intentionally returns the author's current HTML with no-store and sandbox CSP.

Fact: API-token and enrollment files must be regular mode-0600 files; final
symlinks are rejected. Open-handle identity checks and bounded reads enforce
4096 bytes for the token and 1024 bytes for enrollment, including changes after
the initial file inspection. Enrollment requires exactly the known JSON fields,
full 16/32-byte identity/key arrays and valid timezone; duplicates, unknown
fields, alternate key casing, trailing values and malformed Unicode reject.
The separate legacy device listener also admits at most four handshakes; EPN2
already has its own four-worker admission policy and is not a legacy fallback.

Boundary: screen mode retains its loopback default, loopback-only validation
and required `-trusted-html` acknowledgment. Direct optional non-loopback bind
remains pending explicit approval accepted by automatic action review; two
attempts to change that restriction were rejected during this work. No external
listener, firewall rule or proxy was deployed. An operator-managed HTTPS reverse
proxy can expose the existing loopback HTTPS endpoint as a separate deployment
choice: preserve encrypted, certificate-verified upstream transport, application
Bearer authentication, request limits and the author trust model. External
certificate, proxy and firewall configuration require operator acceptance.

Measurement: host regression tests exercise native TLS 1.3 success/TLS 1.2
rejection, half-open connection limits, concurrent mutation limits, ambiguous
headers and unsafe local secret files. They do not prove external deployment
availability or physical Pico/radio/panel acceptance.

Sources: [Go HTTP server and HTTP/2 controls](https://pkg.go.dev/net/http#HTTP2Config),
[simultaneous connection limit](https://pkg.go.dev/golang.org/x/net/netutil#LimitListener),
and [RFC 9110 field order/duplication](https://www.rfc-editor.org/rfc/rfc9110.html#section-5.3).
