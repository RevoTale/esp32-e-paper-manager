# Unified screen session: EPS2 migration design

2026-09-07. D1–D2 codec, retained session, connection adapter and client are
implemented/tested; unified hardware runtime acceptance remains pending.
Reuses the existing tested stream receiver/panel sink and secure record layer.
EPS1 remains the immutable historical hardware checkpoint; production tools and
candidate firmware move together to EPS2, never silently reinterpret EPS1.

## Ownership and evidence

- `engine` / manager: authoring, layout, complete compositing, pixel diff,
  timestamps, cadence, authenticated device enrollment and latest target.
- `screenwire`: checked EPS2 syntax, target capabilities, transaction identity,
	bounded diagnostics. No HTML, GPIO commands, keys, sockets or device discovery.
- `streamrx`: ordered bounded logical passes, digest checks, complete-before-
  visible commit. No full-frame staging. All passes represent identical canonical
  pixels; the panel adapter alone chooses inverse/original controller polarity.
- Session owner: one USB/network writer; physical refresh cannot be preempted.
  It retains last terminal evidence across connection loss, not across reboot.
- Board/HAT adapter: exact pins, safe initialization/power-off, SPI/BUSY stages.
- `securetransport`: existing mutual proof and AEAD records before network input
  reaches parsing/staging. EPS2 CRC detects damage; it is not authentication.

Three diagnostic questions determine the wire/status fields:

1. Did bytes reach the receiver, and which transaction/pass/offset is accepted?
2. Did failure occur in framing/authentication, staging/SPI, or a specific BUSY
   wait? Report stable domain/code/phase/step/command/offset/BUSY-known/value.
3. Did a completed full cycle survive connection loss, or is completion unknown?

No raw HTML, asset bytes, SSID/password/token/key or internal error strings in
diagnostics. Manager structured events correlate namespace + transaction ID and
entry point (USB, authenticated device, HTTP mutation, maintenance). Hardware
success remains protocol evidence, never automatic visible acceptance.

## Framing contract

Preserve the simple existing 32-byte little-endian header shape; change magic to
`EPS2`. Fields: magic[4], kind u8, pass u8, payload length u16, transaction
namespace u64, transaction ID u64, offset u32, CRC32 u32. CRC covers header before
the CRC slot and payload. Reject unknown types/reserved values before dispatch.

Increase the maximum payload to 1024 bytes, with negotiated device maximum <=
1024. Receiver/storage/transmit scratch sizes remain compile-time bounded. This
amortizes stop-and-wait headers/ACKs; measure complete two-pass wire bytes before
accepting D8. A panel sink still writes its own bounded inversion chunks.

EPS2 uses Hello/Acquire/Bind/Begin/Data/Commit/Query/Abort/Reply, with fixed typed
capability and diagnostic bodies rather than arbitrary key/value extensions.
Exact offsets/enums and reply-correlation rules are frozen in
[EPS2 wire contract](../../eps2-wire.md) with independent codec tests.

EPN2 carries exactly **one complete EPS2 record per AEAD envelope**, in both
directions. TCP may split ciphertext arbitrarily; an EPS2 record may not be
split across envelopes or bundled with another. `ReadRecord` validates the
whole envelope before dispatch; the host reply adapter validates a complete
Reply before exposing fragmented bytes to the shared USB-style client. The
first encrypted byte switches from idle waiting to one fixed active-record
deadline. Timing only the first plaintext byte would start too late.

Capabilities must expose physical profile/version, logical dimensions/stride,
color model, supported refresh/encoding flags, passes, max decoded chunk,
minimum cadence/remaining cooldown, public boot identity and provisioned device ID
(all-zero identity means USB-only unenrolled, not a network wildcard).

The baseline encoding is raw canonical mono1, MSB-left, 0 white / 1 black,
zero row padding. Optional negotiated PackBits now has bounded decode, fault,
two-plane replay and measured wire-byte evidence in
[the compression contract](../../eps2-compression.md). No unknown capability
implicitly enables partial refresh or compressed input.

The secure record bound is now decoupled from the historical 284-byte display
protocol: 1,056-byte plaintext / 1,088-byte AEAD envelopes pass maximum-record
and fragmented-read tests. Final TinyGo resource qualification is pending; do not
assume `RecordStream.Write` splits oversize input. The accepted panel sink still
allows only 100 bytes per Write, so a separate bounded chunk adapter splits
authenticated logical records before SPI. Preserve the proven panel row buffer.
The 800×480 adapter negotiates 1,000 bytes (10 × 100): 48,000-byte planes divide
exactly and the recording-panel regression proves the complete SPI/GPIO trace
equals the accepted buffered driver. The adapter allocates no per-Write storage.

## Transaction versus connection identity

- A transaction is immutable `(boot nonce, lease generation, id, final-pixel
  SHA-256)`. Firmware issues a monotonically increasing nonzero boot-scoped
  lease generation; the header's namespace field carries that generation.
  The manager chooses increasing IDs within its lease. Identical-pixel
  maintenance uses a new ID; it must not be mistaken for retry.
- A reconnect uses the same pending transaction identity for reconciliation.
  Network crypto session counters/nonces are separate and always freshly
  authenticated; reusing an update ID never reuses an AEAD nonce/session.
- Firmware supplies `flash UID[8] || big-endian durable boot epoch[8]`, not
  pseudorandom entropy. Reserve and verify the epoch before exposing EPS2 or
  attempting network authentication. A changed/unknown boot identity
  means no retained completion proof. Do not infer it from the e-paper image,
  requested pixel hash, USB port name or provisioned identity alone.
- Query the pending transaction before deciding to resend after a missing
  terminal ACK. Exact identity plus digest plus unchanged boot can prove it
  completed, requiring zero new SPI writes. Same ID/different digest rejects.
- Retain bounded current state, consumed-ID high-water and one last terminal
  record. Begin consumes its ID before sink initialization, including later
  failure/abort. Interrupted staging requires a new ID/full upload; never resume
  a controller write pointer. Non-retained IDs <= high-water return stale/unknown,
  never not-seen or inferred success. Query includes the expected digest.
- Persist terminal evidence before attempting its ACK. Repeated successful
  Commit is zero-SPI. Failed/retired IDs cannot restart. Disconnect clears byte
  framing/incomplete staging, not terminal evidence or the high-water fence.
- Invalidate *current-image equivalence before* invoking physical Commit. A
  later BUSY/power-off failure may follow a visible transition. Historical old
  success can remain true, but only full new success restores current proof.
- Hello is read-only discovery. Reconnect binds the unchanged expected boot and
  lease generation; it does not acquire another lease. Explicit Acquire requires
  the observed boot/current generation as a compare-and-swap and an idle owner,
  increments the generation without wrap, and invalidates old completion reuse.
  Old A -> new B -> replayed A cannot stage. Connection records require a fresh
  boot/lease binding after disconnect; a reboot cannot reuse an old binding.
- Acquire also carries a fresh random 128-bit client claim, retained unchanged
  for retries. Hello/status/errors never disclose another writer's claim.
  A same-boot, non-wrapping `expected generation + 1` plus matching claim allows
  an idempotent grant reply only: no high-water/evidence/cadence/binding reset.
  Reconnect must prove the remembered claim, not adopt a generation from Hello.
  This fences accidental competing owners; it does not replace authentication.
- Bind permits one live connection and issues a non-reusable local serial.
  A stale connection cannot write or disconnect its replacement, even if both
  possessed the same lease. Explicit preemption aborts staging and releases the
  old binding before the next binding; a physical Commit remains synchronous.
- No Hello/Acquire resets cadence or credential-generation fences. USB staging
  preemption is explicit through the owner. Credential change fences the lease
  and completion evidence after safely finishing/aborting the current operation.
- Reboot, unknown outcome or changed credentials requires a full resync. A
  bounded backoff and device cooldown govern it; never a blind refresh loop.

The coordinator must distinguish confirmed scene equivalence from the last
actual delivery ID. Manager process restart likewise discards unproven cache.
No automatic catch-up burst of missed maintenance cycles.

## Lifecycle and composition

USB has priority before a physical refresh starts. It can abort incomplete
network staging and then take ownership; an already-started panel refresh
finishes its BUSY/power-off path before transfer or provisioning proceeds.
Both transports use the same record/receiver/sink core. No second panel owner.

Physical USB provisioning is the only credential-write route. Rotation/erase
aborts incomplete staging, invalidates connection/transaction evidence and
reloads complete config. Blank/invalid config keeps USB functional and Wi-Fi
disabled. A generic UF2 contains no user's key or WLAN secret.

The physical control protocol is EPCQ/EPCR v2, with stable result codes and an
explicit unknown-storage state; the persisted EPC2 journal stays v1. See
[provisioning](../../provisioning.md). Fencing includes unbound authenticated
connections and queued old-auth work, not just the current pixel writer.
`screenlink.Reconfigure` revokes existing connection instances; the unified
owner additionally gates new network opens by its own non-wrapping auth access
generation. Neither fence resets the physical refresh safety floor.

WPA3-SAE only; no WPA2 downgrade or Bluetooth. Pico initiates IPv4 connections
to its provisioned manager; no Pico HTTP endpoint exposed to public traffic.
IPv6 on the MCU remains deferred by the user's clarification; the manager's
external HTTPS listener can serve both IP families. Router exposure and TLS
deployment are operator work, not mutations authorized by firmware development.

Connection roles and nonce lifetime are specified in
[ADR-014](../../../decisions/014-durable-device-sessions.md). Pico is the TCP dialer but
uses the existing cryptographic Device role; the manager supplies OS-generated
randomness. `EPN2` has an exact 20-byte public preface and no legacy fallback.

The full-cycle corner label is manager-rendered in the protected rectangle.
Its candidate cycle-start timestamp is confirmed only after matched terminal
success; failure cannot advance confirmed metadata. It denotes the server's
full-cycle start, not a measurement of the exact electro-optical transition.
Configured timezone comes from USB provisioning/enrollment. Maintenance defaults
to 600 seconds with configuration, independent of optional authoring debounce.
Device safety floors still win over requested frequency.

## Required tests before replacing EPS1 source integration

- Byte-accurate codec and independent malformed/overflow/unknown-field oracles.
- Varied/non-byte-width display profiles; unsupported capabilities fail closed.
- Missing/reordered/duplicate chunks, corruption, both-pass digest mismatch,
  truncation, idle/total deadline and no complete image -> no refresh.
- Drop before/after Commit, lose only terminal ACK, reconnect/new boot/new epoch,
  duplicate Commit and same ID/different hash; prove physical Commit count.
- USB preemption at each staging boundary versus active refresh, credential
  rotation, key rejection, bounded reconnection and no stale completion reuse.
- Diagnostic fixtures for SPI errors and each BUSY phase, with no secret/source
  leakage. An induced fake-panel failure must be explainable from decoded status.
- Real HTTPS -> native renderer -> USB/AEAD codec -> unified owner -> recording
  panel; final logical planes equal preview and use accepted panel polarity.
- Full candidate dependency graph, TinyGo size/stacks, no unexpected frame
  allocation, CGO=0 native manager/tools and actual wire byte/latency reporting.

Primary references: [Waveshare HAT](https://www.waveshare.com/wiki/E-Paper_Driver_HAT),
[pinned panel source](https://github.com/waveshareteam/Pico_ePaper_Code/blob/c9bcd84db5adf5f085353649a8a5c31492bc5fb8/c/lib/e-Paper/EPD_7in5_V2.c),
[Go binary encoding](https://pkg.go.dev/encoding/binary),
[Go AEAD contract](https://pkg.go.dev/crypto/cipher#AEAD).
Exact hardware authority and accepted physical evidence remain in root
`docs/epaper-hardware-sources.md` and `docs/epaper-debugging-history.md`.
