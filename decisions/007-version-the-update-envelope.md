# ADR-007: Version the transport-neutral update envelope

## Status

Accepted.

## Date

2026-09-02.

## Context

USB, the home manager, and the encrypted device link need one deterministic
request/result contract without depending on parser, renderer, runtime, or
panel implementations. The Pico cannot afford hidden copies or an unbounded
general-purpose serializer. Corruption diagnostics and retry identity must be
stable across host and TinyGo builds.

## Decision

Use a fixed version-1 binary envelope with a 160-byte canonical header, bounded
HTML payload, and four-byte trailer. Decode borrows the payload. SHA-256 defines
content identity; independent header and whole-message IEEE CRC-32 values catch
accidental corruption. Typed stages, codes, and bounded limit names represent
diagnostics. A static independent golden vector freezes compatibility.

Keep authentication outside this envelope. Physical USB accepts the envelope
directly. Network records wrap it in the authenticated encrypted device-link
protocol. Hosts resolve the provisioned IANA timezone and send a fixed
`YYYY-MM-DD HH:MM` display value because the Pico does not carry a timezone
database.

## Consequences

- All transports can reuse the contract without importing rendering internals.
- Unknown versions and non-canonical encodings fail closed.
- Payload ownership stays explicit and allocation-light.
- CRC-32 is never treated as proof of authenticity.
- A future schema change needs a new version and golden vector.

## Sources

- https://pkg.go.dev/crypto/sha256
- https://pkg.go.dev/hash/crc32
- https://pkg.go.dev/encoding/binary
- https://tinygo.org/docs/reference/lang-support/stdlib/
