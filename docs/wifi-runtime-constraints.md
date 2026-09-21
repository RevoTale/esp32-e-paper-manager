# Pinned Wi-Fi runtime constraints

2026-09-07 source review. This records implementation requirements, **not**
completed target acceptance. cyw43439 v0.1.1, lneto v0.1.0, PIO v0.2.0,
TinyGo 0.41.1; no dependency source patched or new test skipped.

## Owner isolation and scheduling

One permanent network worker and one permanent packet pump; no goroutine
replacement after a stuck operation. USB/panel owner never calls radio join,
reset, packet I/O or stack helpers. Its cancellation is a nonblocking version
fence. Every queued open/push/close also checks owner access + connection token.
`screenbridge` has one bounded request slot and one response slot. Borrowed
request bytes stay owned by the worker until the owner acknowledges them, even
after cancellation. The main loop must keep servicing the bridge during USB.

`Join` holds the radio mutex. Its nominal ten-second wait checks the clock only
after `check_status` drains pending packets; continuous traffic can prevent
return. `Reset`, `PollOne`, `SendEth`, and `NetFlags` cannot interrupt that mutex.
Do not advertise a bounded total Join duration or wait for it in USB code.

TinyGo tasks are cooperative. Construct through existing `NewPicoWCmdBus`/`New`
APIs; set a PIO timeout and explicitly yield from the CS callback **after** CS
goes high/inactive. Every completed command then yields, including fast success
paths. Default DMA timeout is disabled. `SetTimeout` rounds up internally, so
its configured value is not a measured end-to-end latency guarantee. This
protects owner scheduling; it does not make Join cancellable.

Use locked network-side `NetFlags` and `FlagRunning`; `IsLinkUp` reads mutable
state without locking. Never use radio flags as the owner authorization gate.
In this pinned driver, its local `FlagRunning` constant is `1 << 5`.

## Stack concurrency and cache lifetime

`StackAsync` does not make every method safe: `ResultDHCP` reads and returns
borrowed mutable data without its internal lock. DHCP ingress can update options
even after Bound. Protect ingress, egress, and result extraction/copy/adoption
with a small network-only mutex. Never hold it across radio calls: the radio
receive callback enters the stack while already holding the radio mutex.

Use asynchronous Start/Result operations with cancellation/deadline checks.
Avoid the blocking wrappers' unlocked ARP cleanup and TCP InternalHandler reads:
use public locked `DiscardResolveHardwareAddress6`, `DialTCP` + `Conn.State`.
`Close` starts graceful FIN; `Abort` resets identity and wakes old reads on their
next backoff iteration. Old queued network events remain fenced regardless.

Reset retains some subnet/DNS/client fields. Recreate/quiesce the stack or
explicitly validate fresh DHCP state before dialing. Never silently reuse an
old DNS server when fresh DHCP omits it. Honor lease/renewal deadlines; a cached
address is not valid forever. An absent fresh DNS server may still allow a
provisioned literal IPv4 manager. MCU IPv6 remains explicitly deferred.

## `stacknet` ownership contract

Fact: the worker serializes `Reset`, `Configure`, and `Dial`. Quiesce the packet
pump before `Reset`, including any packet already copied out for radio sending.
`Reset` creates a fresh stack. Call `Configure` once after each successful reset;
after a failed configuration or lease renewal deadline, reset again before
configuration. This is fresh DHCP acquisition, not an in-place renewal protocol.

The worker closes each returned socket and finishes all of its I/O before the
next `Dial` or `Reset`. Returned sockets borrow the client's reused TCP connection
and byte buffers; retaining an old socket into the next attempt is unsupported.
The serialized `screenwifi` attempt lifecycle is responsible for this boundary.
`Abort` is a network-side operation, not an owner-thread cancellation primitive.

Fact: the outer client mutex also covers TCP open/abort/close identity changes.
Pinned `internet.node.IsInvalid` reads the connection-ID pointer without the
TCP connection mutex, so `Conn.Abort` alone is insufficient synchronization with
packet routing. A real concurrent egress/abort test reproduced this race before
the shared outer lock fixed it. Radio calls never execute under that lock.

Fact: pinned TCP active-open remains `StateClosed` until SYN egress. The adapter
uses public locked `State` and `RemotePort` getters to distinguish a pending
active open from a closed connection. It never reads `InternalHandler` directly.
Each completed or failed dial discards its possible on-link ARP query through
the locked public API. Otherwise three failed distinct local dials exhaust the
three-entry upstream query table; aborting TCP does not free those queries.

DHCP result extraction, validation, and adoption occur under the same mutex as
ingress/egress. The validated router is copied once, not reread from later mutable
DHCP results. Lease admission rejects invalid/short leases, non-host subnet
addresses, router/address conflicts, and limited-broadcast DNS/manager addresses.
Both hosts of a `/31` remain permitted. The adapter stops packet I/O at renewal
time (half the lease when T1 is absent/invalid), capped at 24 hours; it does not
claim DHCP renewal, IPv6, DNSSEC, or measured radio availability.

## Packet-level regression evidence

Measurement, 2026-09-07: focused `go test -race -coverprofile=... ./stacknet`
passes with 95.0% statement coverage; `tools/bin/golangci-lint run ./stacknet/...`
reports zero issues. No test skips, linter suppressions, or vendor patches.

Tests exercise actual pinned `lneto` Ethernet/IPv4/UDP, DHCP server, ARP handler,
DNS codec, and TCP stack APIs rather than substituting a stack interface:

- Complete DHCP DORA followed by gateway ARP; copied router identity, omitted
  fresh DNS, invalid lease/MAC, three bounded DHCP attempts, cancellation, and
  renewal expiry rejecting subsequent packet I/O.
- DNS A, AAAA-only, loopback, NXDOMAIN, empty answers and no-response deadline.
- Delayed first SYN, full TCP handshake and payload delivery, cancellation/abort,
  four failed local ARP attempts without slot exhaustion, and concurrent
  packet routing/abort under the race detector.

These are in-memory software measurements. They do not establish real WPA3
association, RF cancellation latency, electrical energy, or visible panel output.

## Primary source anchors

- [Radio constructor](https://github.com/soypat/cyw43439/blob/v0.1.1/bus_pico_pio.go),
  [CS transaction boundary](https://github.com/soypat/cyw43439/blob/v0.1.1/bus.go).
- [Join and state](https://github.com/soypat/cyw43439/blob/v0.1.1/wifi.go),
  [packet draining](https://github.com/soypat/cyw43439/blob/v0.1.1/ioctl.go),
  [network flags/polling](https://github.com/soypat/cyw43439/blob/v0.1.1/netif.go).
- [PIO timeout](https://github.com/tinygo-org/pio/blob/v0.2.0/rp2-pio/piolib/spi3w.go),
  [DMA timeout/abort](https://github.com/tinygo-org/pio/blob/v0.2.0/rp2-pio/piolib/dma.go).
- [Async stack/DHCP](https://github.com/soypat/lneto/blob/v0.1.0/x/xnet/stack-async.go),
  [blocking helpers](https://github.com/soypat/lneto/blob/v0.1.0/x/xnet/stack-blocking.go),
  [TCP lifecycle](https://github.com/soypat/lneto/blob/v0.1.0/tcp/conn.go).
- [Pending active-open state](https://github.com/soypat/lneto/blob/v0.1.0/tcp/handler.go#L422),
  [unlocked routing identity](https://github.com/soypat/lneto/blob/v0.1.0/internet/definitions.go#L195),
  [implicit local ARP query](https://github.com/soypat/lneto/blob/v0.1.0/x/xnet/stack-async.go#L389),
  [DHCP option mutation before state rejection](https://github.com/soypat/lneto/blob/v0.1.0/dhcpv4/client.go#L226).
