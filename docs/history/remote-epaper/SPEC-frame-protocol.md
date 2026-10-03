# Spec: frame-protocol

Status: approved automatically by user policy on 2026-08-30.

## Objective

Define one small, versioned binary protocol for transferring an exact 800x480
1-bit frame over ordered byte streams. The same records are used by USB CDC and
TCP. The protocol detects malformed or corrupted records, never allocates from
wire-provided lengths, and lets the Pico receive directly into its single
48,000-byte staging frame.

This module encodes and decodes records and validates a sequential frame
transaction. It does not own USB/TCP connections, arbitrate USB against Wi-Fi,
authenticate peers, schedule panel refreshes, or drive the panel.

## Assumptions and Non-goals

Assumptions for version 1:

- transports provide an ordered byte stream; datagrams are out of scope;
- the device owns one 48,000-byte staging frame and no second full-frame copy;
- the host converts, rotates, scales, thresholds, and dithers images before
  transfer;
- the device accepts only the panel format defined below;
- a broken stream may restart a frame from byte zero; resume is not required;
- USB priority is a `device-runtime` policy, not a wire-format rule;
- the local network is not trusted. CRC detects accidental corruption but is
  not authentication or protection against deliberate tampering.

Version 1 deliberately omits compression, encryption, authentication, resume,
out-of-order chunks, retransmission windows, datagrams, and partial refresh.
Wi-Fi authentication and exposure are specified by `wifi-transport` before it
is enabled.

## Tech Stack and Evidence

- TinyGo 0.41.1 and Go 1.26.2.
- Standard-library `hash/crc32` with the IEEE polynomial:
  https://pkg.go.dev/hash/crc32
- TinyGo USB CDC currently uses 512-byte RX and TX ring buffers; this is a
  target-version source observation, not a stable public API guarantee. The
  inspected TinyGo 0.41.1 file is `src/machine/usb/cdc/ring.go`, SHA-256
  `60de20ee1918e9fd06aa9f55f3a4915cdbd5ea5ce747ea07e10efcd4c1f94b7e`.
  Recheck it on toolchain upgrades.
- The inspected Go 1.26.2 `src/hash/crc32/crc32.go` has SHA-256
  `85f9b1484d31d449fda4c70b6df4e8c78455b5384d06fecd5f7a359638de2f76`.
- The frame contract comes from `SPEC-panel-driver.md`: 800x480, 100 bytes per
  row, row-major from top-left, most-significant bit first, 0=white, 1=black.

The Go documentation describes IEEE as the most common CRC-32 polynomial and
`Update` as the incremental checksum operation. CRC is used only for accidental
corruption detection. It must never be described as a cryptographic integrity
or authenticity mechanism.

## Commands

Run inside the existing devcontainer from
`/workspaces/pico-sandbox/experiments/03-remote-epaper`:

```sh
gofmt -w protocol/*.go protocol/*_test.go cmd/protocol-check/*.go
go vet ./...
go test ./protocol
go test -run '^$' -fuzz FuzzDecoder -fuzztime=10s ./protocol
tinygo build -target=pico2-w -scheduler=tasks -size=full -o protocol-check.uf2 ./cmd/protocol-check
tinygo build -target=pico2-w -scheduler=tasks -print-allocs=protocol -o protocol-check.uf2 ./cmd/protocol-check
./scripts/check-protocol-resources.sh
```

No Git commit is created unless explicitly requested.

## Project Structure

```text
experiments/03-remote-epaper/
  SPEC-frame-protocol.md
  protocol/
    codec.go              constants, records, encoder, and decoder
    receiver.go           sequential frame transaction validator
    codec_test.go
    receiver_test.go
    fuzz_test.go
  cmd/protocol-check/     TinyGo compile and allocation probe
  cmd/protocol-empty/     identical empty baseline probe
  scripts/check-protocol-resources.sh
```

The `protocol` package imports neither `machine` nor networking packages. Host
and device use the same constants, codec, and transaction rules.

## Frame Contract

```go
const (
	FrameWidth  = 800
	FrameHeight = 480
	FrameStride = 100
	FrameBytes  = 48_000
	PixelMono1  = 1
)
```

Bytes are rows from top to bottom. Within a row, the first pixel is bit 7 of
the first byte. A clear bit is white and a set bit is black. Width, height,
stride, pixel format, and frame length in `BeginFrame` must match these exact
values; version 1 does not negotiate alternate formats.

## Wire Record

All integers are unsigned little-endian. Every record has a fixed 24-byte
header, 0 to 256 payload bytes, and a 4-byte trailer. The largest record is 284
bytes and therefore fits in the inspected 512-byte USB CDC rings with headroom.

| Offset | Size | Field | Rule |
|---:|---:|---|---|
| 0 | 4 | magic | ASCII `EPDR` |
| 4 | 1 | version | exactly `1` |
| 5 | 1 | type | one known record type |
| 6 | 2 | flags | zero in version 1 |
| 8 | 4 | transfer ID | non-zero for frame records |
| 12 | 4 | value | type-specific unsigned value |
| 16 | 2 | payload length | 0 through 256 |
| 18 | 2 | reserved | zero |
| 20 | 4 | header CRC | CRC-32/IEEE over bytes 4 through 19 |
| 24 | N | payload | exactly `payload length` bytes |
| 24+N | 4 | record CRC | CRC-32/IEEE over bytes 4 through 23+N |

`HeaderSize` is 24, `TrailerSize` is 4, `MaxPayload` is 256, and
`MaxRecordSize` is 284. The header CRC validates the payload length before the
decoder waits for payload bytes. The record CRC covers the validated header and
payload. CRC values are placed little-endian even though `hash.Hash32.Sum`
normally emits big-endian bytes; the implementation uses `Sum32`/`Update` and
encodes the integer explicitly.

The decoder uses a fixed `[MaxRecordSize]byte` scratch buffer. It never sizes a
slice, allocates memory, or loops without consuming input based on an
unvalidated length.

## Record Types

```go
type Type uint8

const (
	TypeHello Type = 1 + iota
	TypeBeginFrame
	TypeFrameChunk
	TypeCommitFrame
	TypeCancelFrame
	TypeAck
	TypeError
	TypeStatus
)
```

Direction and payload are exact:

| Type | Direction | `transfer ID` | `value` | Payload |
|---|---|---:|---:|---|
| `Hello` | device to host | 0 | 0 | fixed 16-byte device information |
| `BeginFrame` | host to device | non-zero | 0 | fixed 20-byte frame metadata |
| `FrameChunk` | host to device | active ID | byte offset | 1..256 frame bytes |
| `CommitFrame` | host to device | active ID | 0 | empty |
| `CancelFrame` | host to device | active ID | 1 (host request) | empty |
| `Ack` | device to host | relevant ID | next expected offset | fixed 1-byte acknowledgement kind |
| `Error` | device to host | relevant ID or 0 | error code | fixed 4-byte next expected offset |
| `Status` | device to host | relevant ID or 0 | runtime state | empty |

The 16-byte `Hello` payload is:

| Offset | Size | Field | Required v1 value |
|---:|---:|---|---:|
| 0 | 2 | maximum payload | 256 |
| 2 | 2 | width | 800 |
| 4 | 2 | height | 480 |
| 6 | 2 | stride | 100 |
| 8 | 4 | frame bytes | 48000 |
| 12 | 1 | pixel format | 1 |
| 13 | 1 | protocol minor | 0 |
| 14 | 2 | reserved | 0 |

The 20-byte `BeginFrame` payload is:

| Offset | Size | Field | Required v1 value |
|---:|---:|---|---:|
| 0 | 2 | width | 800 |
| 2 | 2 | height | 480 |
| 4 | 2 | stride | 100 |
| 6 | 1 | pixel format | 1 |
| 7 | 1 | reserved | 0 |
| 8 | 4 | frame bytes | 48000 |
| 12 | 4 | frame CRC-32/IEEE | host-computed checksum |
| 16 | 4 | reserved | 0 |

Unknown type, non-zero flags/reserved fields, wrong direction, wrong fixed
payload length, or unsupported metadata is an error. No free-form strings cross
the wire; the host maps stable numeric codes to messages.

Version 1 assigns these exact one-byte acknowledgement kinds:

| Value | Name |
|---:|---|
| 1 | begin accepted |
| 2 | chunk accepted |
| 3 | commit accepted |
| 4 | cancel accepted |

`CancelFrame.value` has only one legal value: 1, host request. The device resets
the matching transaction and responds with ACK(cancel). Device transport reset
sends nothing; USB preemption is reported to Wi-Fi as `Error(value=9)` after
the runtime abandons that transaction, and shutdown uses `Status(value=6)` when
it is still possible to write. Neither notification is acknowledged.

`Error.value` uses: 1 malformed record, 2 unsupported version or value,
3 invalid state, 4 wrong transfer ID, 5 wrong offset, 6 wrong frame metadata,
7 wrong frame CRC, 8 device busy, 9 preempted, and 10 internal failure.
`Status.value` uses: 1 ready, 2 receiving, 3 queued, 4 refreshing, 5 refresh
completed, and 6 failed. "Refresh completed" means the driver returned
successfully; it does not prove visible pixels changed.

Values 0 and 128..255 are reserved in all three enums. A receiver rejects an
unknown ACK kind or cancel/error value. A host receiving an unknown status
value preserves it numerically and reports "unknown status" so a newer device
does not break an older diagnostic client. `Hello.value` and all currently
reserved capability fields must be zero in version 1.

## Transaction Semantics

Only one receive transaction exists per `Receiver`:

```text
idle --BeginFrame/ACK(begin, next=0)--> receiving
receiving --FrameChunk/ACK(chunk, next=offset+len)--> receiving
receiving --CommitFrame/ACK(commit, next=48000)--> complete
receiving --CancelFrame/ACK(cancel, next=0)--> idle
receiving --protocol or CRC error/ERROR--> idle
complete --AcquireFrame--> leased
complete --DiscardFrame--> idle
leased --ReleaseFrame--> idle
```

Rules:

- transfer ID is host-selected and non-zero. The receiver retains only the
  immediately preceding accepted `(SessionID, transfer ID)` pair, regardless of
  whether it completed, was cancelled, or failed, and rejects that exact pair.
  Reuse is allowed after an intervening accepted transaction or in a new
  session; no unbounded ID history exists;
- `BeginFrame` is accepted only while idle and only with exact frame metadata;
- each chunk offset must equal the next expected offset;
- chunks are non-empty and must not exceed 256 bytes or the frame boundary;
- the receiver copies each accepted chunk once, directly into the caller-owned
  staging frame, and updates the incremental frame CRC. The checksum is exactly
  `crc32.ChecksumIEEE(frame[:48000])`; incremental state starts at zero and is
  updated only with accepted chunks in offset order;
- an ACK is emitted only after the state transition and copy have succeeded;
- the host sends the next record only after the matching ACK (stop-and-wait);
- `CommitFrame` is accepted only after exactly 48,000 bytes and matching frame
  CRC; only then may `device-runtime` treat the frame as complete;
- duplicate, skipped, reordered, stale-ID, or post-commit chunks are rejected;
- stream loss while receiving resets that incomplete transaction. Version 1
  restarts at byte zero rather than retaining ambiguous partial state; a
  complete or leased frame follows the explicit ownership rules below;
- `AcquireFrame` changes `complete` to `leased` and returns the same staging
  slice without copying. `BeginFrame`, `Reset`, cancellation, and all mutation
  are rejected while complete or leased;
- after `AcquireFrame`, `device-runtime` calls `ReleaseFrame` only after
  `Refresh` returns. To coalesce a queued frame before acquisition, it must
  explicitly call `DiscardFrame` before accepting the replacing `BeginFrame`.

Stop-and-wait requires 188 chunk ACKs for a 48,000-byte frame at 256-byte
chunks. This is intentionally the simplest bounded design for the first USB
slice. USB and TCP transfer time, CPU awake time, and radio-on time must be
measured. A windowed version is considered only if those measurements justify
the added RAM and recovery state.

## Public Go Contract

The stable package surface is intentionally narrow:

```go
type Header struct {
	Type       Type
	TransferID uint32
	Value      uint32
}

type Record struct {
	Header  Header
	Payload []byte
}

type Role uint8

const (
	RoleHost Role = 1 + iota
	RoleDevice
)

// SessionID is runtime-local and never appears on the wire.
type SessionID uint32

type Decoder struct { /* role, fixed storage, and parser state */ }

func Encode(dst []byte, sender Role, h Header, payload []byte) (int, error)
func NewDecoder(local Role) (Decoder, error)
func (d *Decoder) Push(src []byte) (consumed int, record Record, ready bool, err error)
func (d *Decoder) Reset()

type Receiver struct { /* caller frame, transfer state, incremental CRC */ }

func NewReceiver(frame []byte) (*Receiver, error)
func (r *Receiver) Apply(session SessionID, record Record) Reply
func (r *Receiver) AbortDecode(session SessionID, failure DecodeFailure) Reply
func (r *Receiver) Complete() bool
func (r *Receiver) AcquireFrame() ([]byte, error)
func (r *Receiver) ReleaseFrame() error
func (r *Receiver) DiscardFrame() error
func (r *Receiver) Reset(session SessionID) error
```

Contract details:

- `Encode` validates that `sender` may send the record type, writes into caller
  storage, and returns `ErrBuffer` before mutation when it is too small;
- `NewDecoder` validates the local role; `Push` rejects any type that the remote
  role is not allowed to send;
- `Decoder.Push` may consume part or all of `src` and returns at most one record;
  callers loop while bytes remain;
- returned payload aliases decoder storage and is valid only until the next
  `Push` or `Reset`; callers consume or copy it immediately;
- for non-empty input, each `Push` satisfies
  `consumed > 0 || ready || err != nil`. `Push(nil)` may return no progress and
  preserves a partial record. A buffered record recovered during
  resynchronization may return `ready=true, consumed=0` exactly once;
- after a decode error, decoder state already contains the bounded
  resynchronization result. The caller may continue with `src[consumed:]` or
  close the transport;
- `NewReceiver` requires exactly `FrameBytes`, retains the slice, and never
  allocates another frame;
- `Receiver` is single-owner and not concurrency-safe; `device-runtime`
  serializes it. A runtime assigns every USB attachment or TCP connection a
  unique non-zero `SessionID`; it is not derived from peer bytes. While
  receiving, only the owning session may apply, cancel, abort, or reset the
  transaction. ID replay follows the single-pair bounded rule above;
- peer-caused semantic failures return an encodable `TypeError` `Reply`, not a
  Go error. Go errors are reserved for constructor and local ownership misuse
  such as release without a lease;
- no goroutine, channel, reflection, `fmt`, logging, `io.Reader`, or hidden
  transport timeout is used by this package;
- errors are sentinel values or small typed errors supporting
  `errors.Is`/`errors.As`; they contain codes and offsets, not wire-provided
  strings.

`Reply` is an allocation-free value with an exact representation:

```go
type Reply struct {
	Type       Type // Ack, Error, or Status only
	TransferID uint32
	Value      uint32
	Data       [4]byte
	DataLen    uint8 // 0, 1, or 4 as required by Type
}

func (r Reply) Encode(dst []byte) (int, error) // always RoleDevice
```

ACK stores its kind in `Data[0]`; `DataLen=1`. Error stores the pre-reset next
expected offset little-endian in `Data`; `DataLen=4`. Status has `DataLen=0`.
`Reply.Encode` rejects every other type/length/value combination before
mutation. Runtime-created statuses use the same type; transaction validation
remains independent of transport writes and backpressure.

`Apply` has these exhaustive semantic results:

| Condition | Reply | ID | next offset | State afterward |
|---|---|---:|---:|---|
| valid begin | ACK(begin) | incoming | 0 | receiving |
| valid chunk | ACK(chunk) | active | new offset | receiving |
| valid commit | ACK(commit) | active | 48000 | complete |
| valid host cancel | ACK(cancel) | active | 0 | idle |
| non-owner session while receiving | Error(busy) | incoming | 0 | unchanged |
| begin while complete/leased | Error(busy) | incoming | 0 | unchanged |
| begin while receiving | Error(invalid state) | active | current | idle |
| bad/reused ID | Error(wrong ID) | active if any, else incoming | current or 0 | receiving resets; other states unchanged |
| wrong chunk offset | Error(wrong offset) | active | pre-reset current | idle |
| wrong metadata | Error(wrong metadata) | incoming | 0 | idle |
| early commit or wrong frame CRC | Error(wrong frame CRC) | active | pre-reset current | idle |
| wrong legal record for receiver | Error(invalid state) | active if any | current or 0 | receiving resets; other states unchanged |

`Apply` validates in this order and stops on the first failure: non-zero and
owning session; receiver state and legal record type; transfer ID/replay;
fixed frame metadata or chunk offset/length; final byte count and frame CRC.
The non-owner row has highest precedence and can never reset another session's
transaction. Rows that reset `receiving` apply only to its owning session.

Structural decoder failures never reach `Apply`:

```go
type DecodeFailure uint8

const (
	DecodeMalformed DecodeFailure = 1
	DecodeUnsupported DecodeFailure = 2
)

type DecodeError struct {
	Failure DecodeFailure
}
```

`errors.As` extracts `DecodeError`. Bad version, unknown type, and wrong
direction classify as `DecodeUnsupported`; non-zero flags/reserved fields, bad
fixed length, header CRC, or record CRC classify as `DecodeMalformed`. The
decoder validates header CRC before treating its ID or length as trustworthy,
but `AbortDecode` never relies on the rejected header: when the calling session
owns a receive transaction, it snapshots the active ID and expected offset,
resets to idle, and returns that ID/offset. While idle, complete, leased, or
owned by another session, it leaves state unchanged and returns ID=0, offset=0.
Runtime must extract the typed failure and call `AbortDecode` for every `Push`
error; this is the only decoder-error path to a wire reply. Any non-DecodeError
from `Push` is a local invariant failure: report `Error(internal)`, do not trust
peer fields, and stop that transport session.

## Resynchronization and Failure Policy

The decoder searches for the four-byte magic using a bounded incremental
matcher. Bytes before magic are harmless framing noise, not a reported error.
It preserves at most the longest suffix that is also a magic prefix, so inputs
such as `EPEPDR` resynchronize correctly.

After magic:

1. read the remaining fixed header;
2. reject an invalid version, type, flags, reserved field, header CRC, or
   payload length before reading a payload;
3. read exactly the validated payload and trailer;
4. validate record CRC;
5. expose one record, or return a typed error and boundedly rescan
   `scratch[1:used]` for a nested magic sequence. Retain the earliest subsequent
   complete magic candidate; only after that candidate fails may a later one be
   considered. If none exists, retain the longest suffix that is a magic
   prefix. This prevents `EPDR` inside a valid following record's payload from
   displacing its real header.

One failed candidate performs at most `MaxRecordSize-1` resynchronization byte
inspections before returning the error; it does not recursively validate a
nested candidate in the same call. Tests use a test-only inspection counter to
enforce this bound.

Every byte is processed a bounded number of times. Invalid input cannot cause
an allocation proportional to its length, an out-of-bounds access, or an
unbounded wait inside the parser. Transport/runtime owns idle deadlines and may
close a TCP session after the first malformed record. USB may reset and
resynchronize because it has no equivalent cheap connection close.

A malformed record reaches `Receiver.AbortDecode` through the runtime algorithm
above. A semantic frame error follows the result table above. A complete or
leased frame is not reset by unrelated malformed input. This fails closed:
partially overwritten staging RAM is never displayed. The prior image remains
physically on the e-paper, but the prior frame is not promised to remain in
Pico RAM.

Ownership methods have this exact state matrix:

| Method | idle | receiving | complete | leased |
|---|---|---|---|---|
| `Complete` | false | false | true | true |
| `AcquireFrame` | error | error | frame, then leased | error |
| `DiscardFrame` | error | error | idle | error |
| `ReleaseFrame` | error | error | error | idle |
| owner `Reset` | no-op | idle | error | error |
| other-session `Reset` | no-op | error | error | error |

The slice returned by `AcquireFrame` is immutable and valid only until the
matching `ReleaseFrame` returns. Runtime and panel driver must not retain or use
it afterward.

## Runtime Integration Constraints

- one decoder and one unique runtime-local `SessionID` exist per live transport
  session;
- `device-runtime` is the only owner of the shared `Receiver` and staging frame;
- USB may preempt an incomplete Wi-Fi transaction by sending Wi-Fi
  `Error(preempted)`, resetting the receiver as its current owner, then
  accepting USB;
- USB does not interrupt a physical refresh because the panel driver requires
  its frame to remain immutable until `Refresh` returns;
- while a frame is complete, queued, acquired, or refreshing, the runtime
  rejects `BeginFrame` unless it first deliberately discards a not-yet-acquired
  queued frame for coalescing;
- an accepted `CommitFrame` means frame integrity is verified and queued, not
  that the panel visibly changed;
- `Status` distinguishes ready, receiving, queued, refreshing, refresh
  completed, and failed. The exact scheduling and 180-second full-refresh
  interval belong to `device-runtime`;
- TCP and USB writers must serialize whole encoded records; interleaving record
  bytes from different producers is forbidden.

## Security and Resource Limits

Trust boundaries are USB input and LAN TCP input. Protected assets are device
availability, bounded RAM, the displayed image, and Wi-Fi airtime.

Required controls:

- fixed 284-byte decoder scratch storage and a fixed 48,000-byte caller frame;
- validate magic, version, type, direction, flags, reserved fields, all fixed
  lengths, offsets, state, and both CRCs;
- no memory allocation or loop bound derived from unchecked input;
- no shell path, filename, URL, format string, or executable instruction is
  carried by the protocol;
- numeric errors disclose no memory or internal implementation details;
- TCP connection count, idle timeout, malformed-record policy, and rate limits
  are mandatory `wifi-transport` decisions;
- USB physical access is not treated as remote authentication;
- CRC is not a defense against a malicious client. Wi-Fi transport remains
  disabled until its separately approved design provides peer authentication,
  replay protection, and cryptographic integrity plus bounded handling of
  unauthenticated connections. A trusted LAN alone is not an authentication
  mechanism.

## Testing

Host tests must cover:

- byte-exact golden vectors for every record type;
- encode/decode round trips for payload lengths 0, 1, 255, and 256;
- every possible split point of a maximum-size record;
- multiple records in one input slice and one record across many slices;
- overlapping and partial magic sequences;
- noise before magic; bad version, type, direction, flags, reserved fields,
  fixed lengths, header CRC, record CRC, transfer IDs, offsets, and frame
  metadata;
- truncation at every byte boundary without panic or false completion;
- exact 48,000-byte success, short commit, overflow, skipped/reordered/
  duplicate chunks, wrong frame CRC, cancel, and reset;
- exact `Reply` bytes and receiver state for every row of the semantic result
  and ownership matrices, including decoder abort while idle, receiving,
  complete, leased, and owned by another session;
- rejection of the immediately repeated `(SessionID, transfer ID)` pair,
  acceptance after an intervening accepted transaction or in a new session,
  and USB/Wi-Fi handoff/preemption;
- every partial record prefix followed by `Push(nil)`, proving state is retained
  without a spin;
- a valid record nested after every possible failure position in a rejected
  candidate, proving bounded resynchronization does not discard it;
- returned payload lifetime documentation and tests that callers consume it
  before the next `Push`;
- `testing.AllocsPerRun` proving zero allocations for steady-state encode,
  decode, and chunk application after construction;
- fuzzing arbitrary decoder input with invariants: no panic, bounded storage,
  consumed count never exceeds input, and progress on non-empty input;
- every one-bit corruption of a golden record, proving no corrupted record is
  accepted; magic corruption may be treated as framing noise;
- CRC conformance vector `123456789` -> `cbf43926` and frame CRC comparison
  against a separately computed golden fixture;
- host/client conformance using the same golden-vector fixtures.

Golden vectors are checked-in byte literals generated independently of
`Encode`; tests must not generate expected bytes with the implementation under
test. A test-only instrumented codec enforces the rescan bound. Test-only copy
and CRC byte counters plus code review establish one copy and one checksum
update per accepted chunk without adding production indirection. Later runtime
integration tests use a fake panel and assert zero refresh calls for every
malformed, interrupted, incomplete, or bad-CRC transfer.

TinyGo checks must prove the package compiles for `pico2-w`, inspect allocation
reports, and record full binary size. An end-to-end USB test sends one frame,
verifies all ACK offsets and final status, then repeats with injected bad CRC,
disconnect, and restart. Transport tests additionally split at 64-byte USB
packet boundaries, wrap nearly-full RX/TX rings, exercise short writes with a
write-all loop and backpressure, reject producer interleaving, and disconnect
during a record or ACK. A later TCP test uses the same vectors and receiver.

## Performance and Energy Acceptance

Before changing stop-and-wait or enabling Wi-Fi by default, record:

- total USB and Wi-Fi frame-transfer time;
- bytes sent in each direction and ACK count;
- Pico awake time during transfer and CRC validation;
- protocol package RAM and flash contribution from TinyGo size output;
- steady-state allocation count;
- Wi-Fi radio-on duration from connection through final ACK.

Acceptance for v1:

- no second 48,000-byte frame;
- decoder scratch storage at most 284 bytes and total protocol static/working
  RAM at most 2 KiB, excluding the caller frame and transport-owned TX buffer;
- incremental protocol flash cost at most 32 KiB against the same empty TinyGo
  probe, with both values recorded rather than inferred;
- zero steady-state heap allocations in codec and receiver;
- every accepted chunk is copied once and checksummed once;
- no polling or goroutine while idle;
- malformed input fails in bounded work and cannot trigger panel refresh;
- an acceptance sample of 20 consecutive USB transfers has zero protocol
  failures and each completes from accepted `BeginFrame` to commit ACK within
  5 seconds;
- Wi-Fi remains disabled by default until an acceptance sample of 20
  authenticated transfers has zero protocol failures, p95 transfer time is at
  most 5 seconds, and connection
  through commit ACK keeps the radio active for at most 10 seconds. TCP Nagle/
  delayed-ACK behavior and whether it can be disabled are recorded with the
  measurement.

`check-protocol-resources.sh` uses the same TinyGo binary, target, scheduler,
optimization, serial mode, and linker flags for `protocol-empty` and
`protocol-check`. The check probe invokes every codec and receiver path through
results consumed by a volatile hardware-visible sink, preventing dead-code
elimination from producing a false pass. The script saves raw `-size=full`,
`-print-stacks`, and `-print-allocs=protocol` output and computes:

```text
static_delta = max(0, static_RAM_check - static_RAM_empty)
stack_delta  = max(0, max_stack_check - max_stack_empty)
RAM_cost     = static_delta + stack_delta
flash_cost   = max(0, flash_check - flash_empty)
```

It rejects `RAM_cost > 2048`, `flash_cost > 32768`, or any reported protocol
heap allocation. It fails closed if an expected field is absent or the TinyGo
output format changes. Raw outputs and computed deltas are retained with the
verification report.

TinyGo 0.41.1 reports only identical `recursive` system-runtime entries for
these no-goroutine probes under `-print-stacks`, without a numeric maximum. In
that exact case the gate records `stack_delta = 0` only when both probes contain
the recursive marker and their linked C-stack reservation is identical; a
numeric entry in either report instead uses the numeric maxima. Missing,
different, or otherwise unrecognized stack evidence fails closed.

If measurement shows ACK latency dominates Wi-Fi energy or transfer time, a
future protocol version may negotiate a bounded credit window. Version 1 wire
semantics are not silently changed.

## Definition of Done

- this specification is approved before implementation;
- wire constants and golden vectors are shared by host and TinyGo builds;
- all host tests, fuzz smoke test, `go vet`, and TinyGo compile/allocation/size
  checks pass in the devcontainer;
- malformed and interrupted transfers never become displayable frames;
- USB and TCP use the same codec and transaction validator;
- measured transfer, allocation, RAM/flash, and radio-time results are recorded
  in `RESEARCH.md` or a dedicated verification report;
- no claim of authentication, visible panel success, or energy improvement is
  made without the corresponding test or measurement.

## Sources

- Go `hash/crc32`: https://pkg.go.dev/hash/crc32
- TinyGo source and version used by the devcontainer:
  https://github.com/tinygo-org/tinygo
- TinyGo Pico 2 W machine documentation (`Serial` is USB CDC):
  https://tinygo.org/docs/reference/microcontrollers/machine/pico2-w/
- Panel frame and lifecycle contract: `SPEC-panel-driver.md`
- Approved boundaries: `CAPABILITY-MAP.md`
