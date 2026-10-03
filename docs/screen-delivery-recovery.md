# Screen delivery ownership and recovery

Status: implemented host/firmware composition, software acceptance only.
Updated: 2026-09-07. No new hardware result is claimed here.

## One scene, one physical cycle

`manager.Screen` owns immutable author revisions and final pixels. `ScreenPump`
alone owns delivery, maintenance and recovery; an EPN2 handshake worker cannot
change scene state. `screendelivery` is the small boundary shared by transports.
`screenhub` accepts one enrolled device with at most four simultaneous handshakes,
one active peer and one queued authenticated replacement. Excess handshakes are
closed; a newer queued peer replaces only the queued peer, not an active request.

The Go manager listens on an OS TCP socket, allowing IPv4/IPv6 where the host is
configured for them. Pico currently connects using IPv4 only. EPN2 authentication
and encrypted records precede EPS2 ownership; there is no legacy-wire fallback.
Public HTTPS authoring remains separately gated until the renderer/input gates
pass. The device-link listener does not authorize public HTML submission.

## Ambiguous completion is not a retry instruction

1. `Send` borrows one frame and retains its transaction identity, not a second
   pixel copy. It confirms only a matching terminal result.
2. An I/O loss preserves the exact physical CycleID in the pump. The hub waits
   for a new authenticated socket, reconnects the retained EPS2 client and queries
   that identity. It never calls `Send` from reconciliation.
3. Confirmed completion resolves the original cycle without another refresh.
   Unconfirmed staging invalidates the baseline and schedules fresh latest HTML.
4. A changed boot/ownership lease requires explicit baseline invalidation and a
   fresh client. Same-boot reconnect with no pending update causes no redraw.
5. Rendering and the corner timestamp happen only after readiness/cooldown, not
   before an arbitrarily long offline wait. Initial/uncertain completion uses a
   conservative negotiated full-refresh cooldown. Rechecking a ready connection
   does not extend that deadline indefinitely.

Maintenance defaults to 600 seconds from confirmed completion. It may bypass
optional author debounce to prevent starvation, but never device cadence. It
uses the latest valid scene, a separate CycleID and an unstamped pixel baseline;
updating the timestamp cannot itself create a pixel-diff update loop.

## Evidence and limits

- Real `net.Pipe` EPN2 authentication + AEAD + EPS2 + recording controller tests
  cover dropped data/Commit replies, reconnect, reboot, exact commit counts,
  profile mismatch, bounded sockets and cancellation. This is not optical or
  physical Wi-Fi acceptance.
- `screenhub`: race suite and lint passed; measured statement coverage 90.2%.
- The manager embeds Go's `time/tzdata`, so minimal container/macOS/Linux builds
  resolve provisioned IANA zones without host packages. The standard library
  documents an approximately 450 KB binary cost; this is manager-only, not Pico.
- TCP cancellation relies on `net.Conn.Close` interrupting blocked I/O; all EPN2
  handshakes and records additionally carry bounded deadlines.

Sources: [Go network connection contract](https://pkg.go.dev/net#Conn),
[embedded zone database](https://pkg.go.dev/time/tzdata),
[AEAD nonce requirements](https://www.rfc-editor.org/rfc/rfc5116.html#section-3.1).
See `eps2-wire.md`, `SPEC-screen-session.md` and `SPEC-engine.md` for wire,
ownership and renderer limits respectively.
