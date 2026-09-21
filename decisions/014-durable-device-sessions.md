# 014. Durable device lifetime, host randomness

2026-09-07. Accepted software contract; production composition/target acceptance
pending. Complements ADR-013 and supersedes weak-RNG assumptions in legacy
`cmd/device`/`network.CounterReader`. No custom TRNG or cryptographic primitive.

## Evidence and decision

TinyGo 0.41.1 `machine_rp2_rng.go` explicitly labels `GetRNG` unsuitable for
cryptography. RP2350 hardware capabilities do not make that API a CSPRNG.
We reuse the existing tested HMAC challenge/AES-GCM record layer, with its
original Device/Host cryptographic roles regardless of TCP connection direction:

1. At boot, reserve and read-verify a nonzero epoch in the two-block tail journal.
   A non-erased invalid record is ambiguous; never fall back to an older epoch.
2. Copy the board's eight-byte flash UID once, before network concurrency, with
   interrupts disabled around `machine.DeviceID()` and the copy. The pinned
   TinyGo RP2350 flash-command implementation exits XIP without an IRQ guard.
   UID is public identification, not entropy, a secret, or global uniqueness.
3. USB boot identity is UID8 + BE64(epoch). One `network.Lifetime` survives all
   network reconnects and credential rotations during that boot.
4. Before any preface bytes, consume a non-wrapping session counter. Pico dials,
   sends `EPN2` + DeviceID16, then uses `ServerHandshake(epoch, counter)`.
5. Manager accepts, validates the exact preface, looks up enrollment and uses
   `HostHandshake` with a fresh 32-byte `crypto/rand` nonce. No weak MCU nonce.
6. Only successful mutual authentication admits EPS2 ownership. Both ends close
   on failed/truncated/replayed handshake. The whole handshake has one 15-second
   deadline, not a fresh timeout per fragment. Subsequent exchange deadlines
   belong to the transport owner; RecordStream is bounded and half-duplex.

DeviceID selection is unauthenticated routing metadata until HMAC verification.
Wi-Fi password, device key, source HTML and raw error strings are never emitted
as diagnostics. Connection/session identity is distinct from EPS2 transaction
identity: a lost terminal ACK may reuse transaction evidence, never crypto state.

## Recovery and limits

Exhausted/damaged epoch storage must not start a network session or fabricate a
normal USB boot identity. Recovery is physical USB only. Credential storage must
be erased and verified before resetting epoch storage; failures stay disabled.
Normal credential erase/rotation does not reset epochs. A journal reset requires
fresh enrollment keys and removal of host-side retained claims/baseline evidence.
This journal protects accidental reuse, not malicious physical flash rollback;
physical USB/flash access remains a trusted provisioning boundary.

Software tests cover exact boot bytes, counter exhaustion, torn/damaged journal,
consumption before a failed attempt, both encrypted directions, all truncated
handshake lengths, wrong keys, prior-session proof replay and deadline failure.
No target flash or electrical/security certification is implied.

## Primary references

- [RFC 5116, nonce requirements](https://www.rfc-editor.org/rfc/rfc5116.html#section-3.1).
- [Go crypto/rand](https://pkg.go.dev/crypto/rand).
- [TinyGo RP2 RNG, v0.41.1](https://github.com/tinygo-org/tinygo/blob/v0.41.1/src/machine/machine_rp2_rng.go).
- [TinyGo DeviceID/flash, v0.41.1](https://github.com/tinygo-org/tinygo/blob/v0.41.1/src/machine/machine_rp2_flash.go).
- [TinyGo RP2350 ROM/XIP, v0.41.1](https://github.com/tinygo-org/tinygo/blob/v0.41.1/src/machine/machine_rp2350_rom.go).
- [Pico 2 W target](https://tinygo.org/docs/reference/microcontrollers/pico2-w/).
