# EPS2 binary screen protocol

2026-09-07. Candidate contract; codec/session tests are implemented, unified
transport and physical acceptance remain pending. EPS1 is historical, not a
fallback. `screenwire` owns syntax; `streamsession` owns transaction evidence;
the connection adapter owns framing/binding; the physical adapter owns SPI.

All integers are little-endian. Header is exactly 32 bytes:

| Offset | Bytes | Meaning |
| --- | --- | --- |
| 0 | 4 | ASCII `EPS2` |
| 4 | 1 | operation |
| 5 | 1 | logical pass; zero except Data/DataPacked |
| 6 | 2 | payload length, maximum 1024 |
| 8 | 8 | boot-scoped lease generation |
| 16 | 8 | transaction ID |
| 24 | 4 | decoded logical byte offset; zero except Data/DataPacked |
| 28 | 4 | IEEE CRC32 over header bytes 0–27 followed by payload |

| Operation | Value | Generation / ID | Payload |
| --- | --- | --- | --- |
| Hello | 1 | zero / zero | empty; read-only discovery |
| Acquire | 2 | observed generation / zero | boot[16], fresh client claim[16] |
| Bind | 3 | granted generation / zero | remembered boot[16], claim[16] |
| Begin | 4 | nonzero / nonzero | final logical image SHA-256[32] |
| Data | 5 | nonzero / nonzero | 1..negotiated maximum bytes |
| Commit | 6 | nonzero / nonzero | same SHA-256[32] |
| Query | 7 | nonzero / nonzero | expected SHA-256[32] |
| Abort | 8 | bound generation / zero | empty; stop incomplete staging |
| Reply | 9 | echo request generation / ID | status[48], plus capabilities[40] for Hello or health[8] for Health |
| Health | 10 | zero / zero | empty; cached network health, no ownership acquisition |
| DataPacked | 11 | nonzero / nonzero | u16 decoded length + PackBits stream, only if feature bit 2 is advertised |
| PanelTrace | 12 | zero / zero | empty; cached last physical cycle, no lease acquisition or Tick |

Packed payload is 4..1024 bytes; decoded length is 1..negotiated maximum.
The exact restricted codec, failure lifecycle and measured wire savings are in
[bounded compression](eps2-compression.md). Progress and SHA always refer to
canonical decoded bytes, not the compressed representation.

Unknown values, reserved bits, incorrect field combinations, lengths, CRC or
noncanonical row padding reject. Header shape is checked before reading the
payload. Zero-copy decode aliases its record buffer; consumers must finish
before reuse. Encode payload must not alias its destination. CRC is damage
detection only: authenticate/decrypt each complete network record first.

## Reply status: 48 bytes

| Offset | Bytes | Meaning |
| --- | --- | --- |
| 0 | 1 | echoed request operation |
| 1 | 1 | stable error code below |
| 2 | 1 | Idle=0, Receiving=1, Ready=2, Complete=3, Failed=4, Closed=5 |
| 3 | 1 | next logical pass (0..2) |
| 4 | 4 | next logical byte offset |
| 8 | 1 | current-image proof, 0 or 1; only Complete can be 1 |
| 9 | 1 | diagnostic domain: none=0, panel=1 |
| 10 | 1 | domain-specific diagnostic code |
| 11 | 1 | physical phase |
| 12 | 1 | physical step |
| 13 | 1 | controller command |
| 14 | 1 | bit 0 BUSY known; bit 1 BUSY pin HIGH (requires known) |
| 15 | 1 | reserved zero |
| 16 | 4 | signed diagnostic byte offset; -1 unknown |
| 20 | 4 | remaining full-refresh cooldown, milliseconds rounded upward |
| 24 | 8 | current device-issued generation, not necessarily the request's |
| 32 | 16 | fresh boot nonce |

Error codes: OK=0, record=1, lease/binding=2, busy=3, stale/evicted=4,
digest identity conflict=5, cooldown=6, receiver state=7, chunk=8,
pass digest=9, deadline/clock=10, hardware=11, configuration=12.
The none diagnostic is entirely zero. Profile 1 diagnostic codes are unknown=1,
BUSY timeout=2, SPI write=3, configuration=4, frame size=5, driver in use=6.
Phase/step values correspond to the existing tested `panel.Phase`/`panel.Step`
enums and are mapped by `paneldiag`; another physical profile owns its mapping.
Never serialize `error.Error()`, source text or secrets.
An error ACK is not evidence that no physical change happened.
Ready/Complete require at least one completed pass and offset zero; Idle has
zero pass/offset; Receiving cannot have completed both passes. The client also
checks exact negotiated pass count and frame bounds. Generic Decode alone is
insufficient: clients call ParseReply for operation/header/length correlation.

## Hello capabilities: additional 40 bytes

| Offset | Bytes | Meaning |
| --- | --- | --- |
| 0, 2, 4, 6 | 2 each | width, height, byte stride, maximum logical chunk |
| 8, 9 | 1 each | passes (1..2), format (mono1=1) |
| 10 | 2 | features: bit 0 raw mono1, bit 1 full refresh (required); optional bit 2 PackBits |
| 12 | 4 | physical profile ID, nonzero |
| 16 | 2 | profile version, nonzero |
| 18 | 2 | reserved zero |
| 20 | 4 | minimum full-refresh interval in milliseconds, nonzero |
| 24 | 16 | provisioned device ID; zero means unenrolled USB-only |

Stride must equal ceil(width/8). Mono1 is MSB-left, 1 black, zero padding.
The HAT V2 profile uses two identical logical passes, 800×480, stride 100 and
maximum chunk 1000; the panel adapter splits these into proven 100-byte SPI
writes and owns plane polarity. Other dimensions require another genuine local
adapter; merely changing advertised numbers cannot adapt a physical driver.

No partial refresh or compression is advertised until separately qualified.
Viewport/layout adaptation happens on the manager after capability negotiation.

## Health: additional 8 bytes

Health is additive: every existing request value and Reply=9 remain unchanged.
Its exact reply payload is 56 bytes; Hello remains 88, other replies remain 48.
Clients reject unknown versions, states, request operations, and wrong body
lengths. The Health request has zero generation, ID, pass, offset, and payload.

| Offset | Bytes | Meaning |
| --- | --- | --- |
| 0 | 1 | health schema version, exactly 1 |
| 1 | 1 | current network state, 0..7 below |
| 2 | 1 | last unexpected failed attempt's stage, 0 means none |
| 3 | 1 | unexpected failed attempts this boot, saturating at 255 |
| 4 | 4 | monotonic runtime uptime seconds, saturating at 2^32−1 |

Network states: disabled=0, joining=1, addressing=2, dialing=3,
authenticating=4, online=5, backoff=6, optional setup failed=7. A failure stage is
captured before Backoff replaces the current state and survives later success.
Expected USB/configuration cancellation is not counted as a failure. This count
is distinct from the private retry-delay counter; it does not reset on Online.

The serialized device owner reads one atomic cached state/stage/count snapshot
plus its elapsed uptime. It does not initialize/join the radio, take a network
lock, acquire/bind a screen lease, or run panel Tick. The generic reply status
carries boot/generation and an idle, non-image-proof result; Health is not a
Query for a physical transaction. Missing health-provider configuration returns
CodeConfig with a valid version-1 body, not fabricated successful health.

Opening USB still invokes its existing physical priority and may abort another
transport's incomplete staging; independent owner Tick still enforces expiry.
Health does not disable these safety rules. See [device health](device-health.md)
for the read-only boundary and software evidence.

## Ownership and retry

### Additive panel trace (2026-09-07)

PanelTrace replies contain status[48] + trace[36] (84 payload bytes). Existing
operation values and lengths do not change. Unsupported firmware is an explicit
request error, never inferred to have an empty successful trace. Use this optional
request only when diagnosing a compatible firmware; ordinary send/Health paths
do not automatically issue it. Missing provider returns CodeConfig with the
canonical unavailable body. The existing authenticated network boundary applies;
USB remains trusted physical access, including its existing priority effects.

Trace offsets: version=1 at0; state at1 (unavailable0, active1, completed2,
stopped/error3); profile-specific phase/step at2/3; saturating boot-local cycle
counter u32 at4; total cycle elapsed milliseconds u32 at8; three pairs of u32
`samples, low_samples` at12,20,28 for power-on, refresh, power-off. All integers
are LE. Unavailable has every other byte zero; other states require cycle>0.
Reject low_samples>samples, invalid version/state and incorrect lengths.

The adapter forwards each existing BUSY read once and records it only within
an existing wait. It does not sample during fixed command delays, add polling,
require a LOW edge, retry, or change SPI/GPIO commands. Zero LOW samples means
only **no LOW read in that wait**, not no controller activity. Samples on timeout
are retained even without BusyDone. Total elapsed includes initialization and
upload; it is cached when the cycle stops, zero while active, not per-stage time.
Clock regression clamps to0 and counters/duration saturate rather than wrap.

One serialized owner updates/reads this bounded RAM-only snapshot. A new cycle
replaces it; abort preserves completed evidence, reboot clears it. It is not
transaction reconciliation, a delivered-power measurement, proof of SPI receipt,
or optical readback. Query with retained lease/ID/digest still owns transaction
evidence. CLI `epaperscreen --panel-status SERIAL_PORT` prints no private IDs,
claims, digest, token, frame content or error text from the device.

The full lease is boot + device-issued generation + private client claim.
Hello never grants ownership. Acquire is idle-owner compare-and-swap; an exact
lost-ACK retry returns the original grant without resetting any state. A second
claim cannot adopt that grant. Bind permits one live writer and has a local,
non-reusable connection serial; stale disconnects cannot affect replacements.

Begin consumes ID before hardware initialization. Failure or interrupted input
requires a new ID and complete upload. Retain one terminal result and a consumed
high-water mark. Query after lost Commit ACK before any retry: unchanged lease,
ID and digest plus Complete/current proof permits zero additional SPI. Unknown
boot/lease or evicted evidence requires new full resynchronization, respecting
cooldown. AEAD nonces/counters always belong to the fresh crypto connection,
never to screen IDs. See [session lifecycle](history/remote-epaper/SPEC-screen-session.md).

`screenclient` retains pending acquisition/update identity across Connect calls.
Lost Acquire ACK permits Hello reporting G+1 only through retrying the original
claim, never adoption. A competing winner returns explicit ErrResync (also a
typed lease rejection), not an indefinitely retryable connection failure.
Reconciliation of active staging requires a successful Closed Abort ACK before
pending identity is released. The client never retries a physical Send itself.

Sources: [Go binary encoding](https://pkg.go.dev/encoding/binary),
[IEEE CRC32](https://pkg.go.dev/hash/crc32),
[Go AEAD contract](https://pkg.go.dev/crypto/cipher#AEAD).
This is a bounded project protocol, not an assertion of an external standard.
