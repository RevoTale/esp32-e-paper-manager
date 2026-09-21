# Terminal ACK-loss acceptance

This is a test-only host harness, not a firmware or production API switch.
`screenhub.TestCommitACKAcceptance` runs against the encrypted Go simulator by
default. Its explicit live mode uses the same EPS2 client and authenticated
EPN2 transport against the enrolled 800x480 profile1/version1 device.

The harness sends one full four-quadrant frame. It consumes and validates the
terminal Commit reply, closes that socket, and returns unexpected EOF to the
client instead of exposing the ACK. The same client then reconnects and queries
the pending identity. Success requires confirmed reconciliation, one initial
Commit, zero reconnect Commits and one Query. The simulator also checks its
physical sink counter equals one.

This proves host-side loss of an authenticated terminal response. It does not
simulate loss before the controller finishes, staging interruption, power loss,
kernel packet drops or process-crash persistence. The hardware has no independent
refresh-counter telemetry, so do not turn request counts into a measured panel
refresh count. Observe the displayed pattern separately.

## Run

Build inside the existing Dev Container from `experiments/03-remote-epaper`:

```sh
go test -race ./screenhub
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go test -c \
  -o ../12-esp32-epaper/.local-mac/epaper-ack-check ./screenhub
```

On the explicitly authorized Mac, first ensure the manager has no in-flight
frame, then stop only that manager. Keep the board powered and USB serial
closed. An operator must be available to remove power on a hardware/unknown
refresh fault. Use the existing provisioned manager address and enrollment:

```sh
.local-mac/epaper-ack-check -test.run '^TestCommitACKAcceptance$' \
  -test.v -test.timeout 9m \
  -epaper-live-enrollment /PRIVATE/device.json \
  -epaper-live-listen MANAGER_IP:9757 -epaper-live-confirm
```

All three live options are required. The harness waits at least180s before its
one full frame and bounds each connection accept to60s. Reconnection allows at
most three transient connection failures while retaining the same client and
pending transaction, following the hub's existing transient-error classification.
Authentication, protocol and lease errors fail closed. It never provisions,
retries Send, or erases flash. On any failure stop and preserve the result;
do not repeat the command blindly. Do not reset/reopen USB during panel work.
After known completion, the ordinary manager can resume with the same keys.

References: project `screenclient/reconcile.go`, `screenpeer/host_screen.go`,
`screenwire/reply.go`, and receiver [safety rules](README.md#safety-and-ownership).
Go stream contract: https://pkg.go.dev/io#ReadFull;
connection shutdown contract: https://pkg.go.dev/net#Conn.

## Staging interruption variant

Select `-test.run '^TestStagingInterruptionAcceptance$'` with the same three
explicit live options. This variant prepares a black frame, validates the
first Data ACK while state is Receiving, then closes the socket before any
Commit. Reconciliation must be unconfirmed without hardware error and clear
the client's pending identity. The simulator asserts zero physical Commits.
The existing visible pattern must remain; there is no automatic follow-up Send.
Cleanup failure requires the operator to remove power, not a software reset.

2026-09-21 live staging run passed in185.42s on the same priority candidate:
zero original/reconnect Commits, one Query, unconfirmed code0/state5(Closed),
pending client cleared. The user confirmed the previous quadrant image remained
unchanged. This is not a power-loss or mid-refresh interruption test.

## Terminal ACK result

2026-09-21, ESP32 Driver Board Rev3 +7.5 V2, application MD5
`708b61f49df59df1f97933dca08c2aff`: live probe passed in242.99s with one Commit,
zero reconnect Commits, one Query and confirmed reconciliation. The user also
confirmed the four-quadrant pattern visible. This observation is not independent
refresh-counter telemetry. An earlier probe stopped at a reconnect TCP
reset; that failed run is preserved in the debugging history. Production
firmware was not modified for fault injection.
