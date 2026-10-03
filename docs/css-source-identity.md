# Checked Go/Blitz CSS identity

Historical cross-parser boundary, superseded 2026-09-07: BZR source manifests
and the Go/Rust adapter were removed. A single native HTML5 tree owns inline
declarations under [SPEC-engine.md](history/remote-epaper/SPEC-engine.md). Commands below require
checkpoint `37a1471`; [the removal map](blitz-removal-map.md) records the migration.

Implemented and host-verified, 2026-09-06. This increment connects the local DOM source
inventory to the manager manifest. It does not activate native/fallback policy
or change the Pico protocol. Existing raw probes remain trusted-only.

## Boundary and format

Go prepares one canonical HTML tree, records stylesheet/inline CSS with a
one-based element preorder ordinal, then serializes that tree. The worker must
verify every manifest source against the actual DOM before layout/paint:
ordered source count, kind, element ordinal and exact decoded CSS bytes.
Duplicate CSS text does not permit matching by content alone. Missing, extra,
reordered or misattributed sources reject the update, never return a frame.

Use local IPC `BZR4`: the existing 24-byte BZR3 header, permitting zero CSS
sources, plus 12-byte source headers (kind u8, three zero reserved bytes,
CSS length u32, element ordinal u32), followed by CSS bytes. Other fields and
asset records retain their existing encoding. Fixed limits remain unchanged;
ordinals are 1–4096. Go emits BZR4 even for zero styles, so an omitted manifest
cannot silently fall back to an unchecked request. Keep BZR1/2/3 compatibility
for existing local probes; do not silently retry those formats on BZR4 failure.

Grammar validation remains before DOM construction. Identity checking is
after DOM construction and before image injection, style resolution or painting
in the same document. It cannot make parsing itself a safe untrusted-input
boundary. Public-input isolation and full capability policy remain separate.
Font registration precedes DOM construction. Identity covers only the CSS
manifest, not full DOM equivalence, resource authorization or rendering policy.

`Job::validate()` returns an immutable `ValidatedJob` borrow. Its render method
prepares one document, verifies BZR4 identity, then moves that document into
painting. The stateless verifier uses bounded shared tree/source discovery but
does not apply bitmap markers or consume native matching budgets. Legacy
BZR1/2/3 jobs remain trusted probes without identity verification; the validated
borrow does not upgrade their guarantees.

CSS grammar errors retain BZE1 codes 1–10 and source-local byte spans. An
identity/context mismatch is an internal adapter/renderer inconsistency: static
`CSS_IDENTITY` failure, no pixels, no raw source or dependency error text. It is
not a retryable transport error or a client assertion of successful rendering.
The Go worker reports its existing fatal ErrWorker path for this mismatch.
This is deliberate until parser differences have a validated user-facing map;
do not reinterpret a mismatch as a normal capability rejection.

## Verification

- Real DOM: empty manifests, implied nodes, sheets plus same-node inline CSS,
  entity decoding, duplicate text on different elements, parser-repaired trees.
- Every mismatch (count, kind, bytes, ordinal, order) fails the whole request.
- Wire: exact bounds, every truncation, reserved fields, source ordinals, legacy
  compatibility and no fallback to an unchecked version.
- Same worker document passes identity then renders known pixels. Malicious
  manifests cannot produce output; valid broad grammar remains accepted.
- Real Go adapter → rebuilt worker preview, plus Rust/Go gates and review.

Fourteen added Rust tests pass (6 identity, 3 wire, 5 real process). Full Rust
suite: 131 passes, zero failures, the same 3 approved engine ignores. Independent
review found no required remaining defect. See `../VERIFICATION.md` for gates.
Go tests also pin serialization stability for implied paragraphs, foster
parenting, adoption-agency repair and scripting-disabled noscript content.

From the experiment directory inside the existing Dev Container:

```sh
go test ./blitzworker ./manager ./renderdiag
go run ./cmd/blitz-preview ./blitz-probe/target/debug/epaper-blitz-probe ./testdata/blitz-identity.html ./build/blitz-identity.png
go run ./cmd/blitz-preview ./blitz-probe/target/debug/epaper-blitz-probe ./testdata/blitz-image.html ./build/blitz-identity-image.png
```

Both actual Go → worker previews pass and were visually inspected. CSS matching
does not assert full typography/layout equivalence. No USB delivery or new
physical acceptance; S5 remains the last accepted panel checkpoint.

## Sources

- [x/net HTML security considerations](https://pkg.go.dev/golang.org/x/net/html#hdr-Security_Considerations): reserialize the parsed tree used for trust decisions.
- [Pinned Blitz HTML sink](https://github.com/DioxusLabs/blitz/blob/67edf2061121382b3e43af19977fc3472aa7e069/packages/blitz-html/src/html_sink.rs): HTML/XML selection and scripting-disabled parsing.
- [Pinned Blitz stylesheet processing](https://github.com/DioxusLabs/blitz/blob/67edf2061121382b3e43af19977fc3472aa7e069/packages/blitz-dom/src/document.rs#L1135): RAWTEXT entity decoding guard.
- Local contracts: `css-style-bindings.md`, `blitz-checked-ipc.md`.

Exact installed beta.2 sources were read; docs.rs retrieval failed during this
increment. No dependencies changed. The ordinal adds four bytes per source
(at most 1024 bytes) to local IPC, not to the Pico link. No measured energy or
peak-memory improvement is claimed.
