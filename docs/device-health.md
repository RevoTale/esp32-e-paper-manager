# Read-only device network health

The additive EPS2 `Health` request reports the cached network worker stage and
retained failure stage without asking the radio or panel to perform work. It is
an operational snapshot, not a watchdog, connectivity probe, optical acceptance
result, or a promise that a stalled upstream radio call will finish.

## API and ownership

`screenclient.ReadHealth(stream)` sends exactly one Health record and parses its
exact correlated response. It does not send Hello, Acquire, Bind, Begin, or
Commit, and does not retry. The caller serializes the borrowed stream and
enforces deadlines; network streams must already be authenticated. Existing
wire values and reply forms are unchanged; older firmware that does not know
Health rejects it without an automatic legacy downgrade.

The board composition installs `Device.SetHealth` before owner/network tasks
start. The callback reads `screenwifi.HealthSnapshot` and `time.Since(started)`;
it does not call Init, Join, reset, DHCP, socket methods, GPIO, or SPI. A missing
worker after optional network construction failure reports state 7 while the
normal USB owner remains usable. Unsafe boot-lifetime recovery still exposes
only physical provisioning control, not a fabricated EPS2 identity or Health
reply. The configuration is not mutated to make health available.

State, last failure, and failure count share one `atomic.Uint32`. Only the
permanent worker writes it; the owner reads one atomic snapshot without radio
or stack locks. The previous state is captured before switching to Backoff.
Unexpected failures count cumulatively in this runtime and saturate at 255;
expected owner/configuration cancellation does not count. Later Online does
not erase the retained failure stage. This is separate from exponential retry
bookkeeping. The synchronization uses Go's
[atomic load/store contract](https://pkg.go.dev/sync/atomic#Uint32), not three
independent field reads that could describe different attempts.

Uptime counts whole seconds since normal runtime composition began, not wall
time or a precise power-on instant. It saturates at 2^32−1 seconds rather than
wrapping. Its source is the owner's
[monotonic elapsed clock](https://pkg.go.dev/time#hdr-Monotonic_Clocks), and the
snapshot does not assert a transactional relationship between uptime and the
network stage. See [EPS2 field layout and numeric states](eps2-wire.md).

## Read-only boundary

A valid Health record bypasses `screenlink`'s per-operation Tick. This matters:
Tick can abort an expired staged image through the physical sink. The direct
Health exchange neither changes the screen generation/binding nor triggers
that expiry. The normal owner keeps polling Tick independently for safety.

Opening a physical USB session still asserts its established priority. DTR can
preempt Wi-Fi and abort incomplete staging before the first Health byte arrives.
Thus a USB status command promises **no ownership/upload/refresh commands**, not
zero system-wide hardware effects while Wi-Fi is staging. Framing/I/O failure
cleanup and independent expiry also keep their existing rules. No panel power
sequence or USB arbitration policy was changed for diagnostics.

## Software evidence

- Wire tests pin old operations 1..9, new Health=10, independent literal bytes,
  exact request/reply shapes, version/state bounds, and Reply-not-a-request.
- A real `screenlink` sink test leaves an expired Ready transaction, its lease,
  its active writer, and all Begin/Write/Commit/Abort counts unchanged through
  an unbound Health query. A subsequent ordinary Tick performs the expected
  timeout/Abort, proving the fixture was actually expired.
- The client test accounts for exactly one 32-byte request plus one 88-byte
  reply, without acquisition or pixel work. Missing provider and stream fail
  explicitly; invalid provider bytes are never serialized.
- Worker tests cover retained pre-Backoff cause, setup failure, cancellation
  exclusion, saturated count/uptime, and concurrent atomic reads under `-race`.

The focused race gate passed for screenwire, screenlink, screenclient, and
screenwifi; strict package lint reported zero issues. This is not target radio,
USB electrical, or visible panel acceptance. No physical query or flash was
performed for this increment.
