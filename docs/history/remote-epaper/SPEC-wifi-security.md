# Spec: authenticated Wi-Fi transport

> Historical foundation. The active migration uses WPA3-SAE only, physical USB
> journal provisioning and Pico-initiated EPN2/EPS2, not the legacy WPA2/ldflags
> composition below. See [screen sessions](SPEC-screen-session.md) and
> [durable-session ADR](decisions/014-durable-device-sessions.md).

Status: approved automatically by user policy on 2026-08-30.

## Scope and dependencies

- Wi-Fi uses `github.com/soypat/cyw43439` with `lneto`: this is the driver
  linked by TinyGo's official Pico W/Pico 2 W documentation. Versions are
  pinned; the firmware build and memory gate must pass before acceptance.
- The radio remains disabled unless SSID, WPA2 passphrase, 32-byte transport
  key, device ID, and durable boot-counter storage are all valid.
- Secrets enter only through TinyGo `-ldflags -X`; no defaults or credentials
  are committed. Build logs must not echo the values. TinyGo's documented
  ThinLTO secret-cache caveat is included in the operator guide.

## Threat model and security contract

- Adversary can observe, alter, replay, delay, and open TCP connections from
  the local network. Physical flash extraction and a compromised host are out
  of scope.
- One TCP connection is active at a time. Unauthenticated connections receive
  no protocol details and are closed within 3 seconds or 512 received bytes.
- RP2350's TinyGo `machine.GetRNG` is explicitly not cryptographically secure.
  It must never generate a nonce.
- Before enabling Wi-Fi, the device durably reserves a never-reused boot epoch.
  Failure, exhaustion, or corruption disables Wi-Fi. A per-boot session number
  creates a unique server challenge for every connection.
- Mutual proof and key derivation use HMAC-SHA-256 with domain-separated
  labels, device ID, boot epoch, session number, and a 32-byte host-generated
  client nonce.
- After authentication, every protocol record is protected with AES-256-GCM.
  Direction and strictly increasing sequence number define unique nonces and
  are authenticated. Duplicate, skipped, malformed, or bad-tag envelopes close
  the connection without changing runtime state.
- Maximum plaintext is the existing 284-byte protocol record. Encrypted input,
  parser buffers, connection count, and deadlines are statically bounded.
- USB has priority: a USB session cancels an incomplete Wi-Fi lease. A panel
  refresh already in progress is never interrupted.

## Discovery and client behavior

- Default TCP port is 9757. Initial release accepts an explicit IP/host only;
  unauthenticated broadcast discovery is not exposed.
- The host parser accepts a DNS name, IPv4 address, or IPv6 literal, adds port
  9757 when absent, and requires an explicit numeric port in 1..65535 when one
  is present. Empty hosts, service names, malformed brackets, and ambiguous
  `host:` forms fail before dialing.
- `epaperctl` chooses USB when `-transport auto` finds it. TCP is used only when
  explicitly selected or when no USB device exists and a host plus key is
  supplied.
- Transport keys are read from an explicitly named environment variable or a
  mode-0600 file; they are never accepted as a command-line value.

## Verification gates

- Independent host tests cover mutual authentication, key separation,
  encryption, both directions, replay, ordering, corruption, timeout, bounds,
  counter corruption/exhaustion, and USB lease preemption.
- TinyGo build reports flash/RAM/stack deltas for Wi-Fi enabled and disabled
  firmware. Twenty authenticated transfers and adversarial recovery cases run
  on the physical Pico 2 W.

Sources:

- https://tinygo.org/docs/reference/microcontrollers/featured/pico-w/
- https://tinygo.org/docs/reference/usage/important-options/
- https://tinygo.org/docs/guides/tips-n-tricks/
- installed TinyGo 0.41.1 `machine/machine_rp2_rng.go`
- https://pkg.go.dev/crypto/hmac
- https://pkg.go.dev/crypto/cipher#NewGCM
