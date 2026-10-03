# ESP32 USB correlation v3

Status: implementation candidate, not hardware-accepted. Replaces unsafe
response-boundary assumptions, not the stored EPC2 configuration or EPN2 link.

- EPCQ/EPCR remain 512 bytes. Version byte4 is3. Operations and metadata retain
  v2 offsets. Request bytes486..501 contain a fresh nonzero random128-bit request
  ID; bytes502..507 stay zero. Response bytes392..407 echo that ID; byte391 and
  bytes408..507 stay zero. CRC32 at508..511 covers the first508 bytes, including
  the ID. The request ID is not secret, authentication or an idempotency key.
- Host sends an operation once, then scans a bounded stream (maximum4096 bytes)
  for a complete canonical response with matching ID and operation. Serial
  polling must be bounded; stale v2/v3 replies cannot confirm a new operation.
  CRC/ID do not protect against a malicious physical USB peer.
- Timeout or exhaustion after a mutation is still an unknown outcome. Preserve
  pending enrollment, poison the connection and never resend automatically.
- Native receiver continues accepting canonical v2 requests for old tools.
  New correlated tools must not fall back to v2 mutations if v3 is unsupported.
  Pico codec/client and device behavior remain unchanged.
- Full-record timeout and idle-byte-loss fix remain separate concerns. No
  display update, flash-layout change or Wi-Fi key rotation is required by v3.

Evidence: debugging history, 2026-09-21: valid EPCR at offset256 within768 bytes,
then a clean512-byte response. Prefix origin remains unknown.

Required tests: leading garbage, stale valid replies, wrong IDs/operations,
CRC/padding corruption, bounded exhaustion, fragmentation, one-write mutation,
poisoned-client reuse, v2 compatibility, and Go-to-native interoperability.

## Candidate use and compatibility

The CLI opts in with `-target esp32 -usb-v3 -port PORT inspect`. Without
`-usb-v3`, existing v2 behavior remains unchanged. Install compatible firmware
before opting in; never use a mutation to discover protocol support. Read polls
are 500 ms, with a 10-second response budget and at most 20 empty reads. A
blocking transport must enforce its own read deadline; the client cannot cancel
an arbitrary blocking `io.Reader`.

The separate macOS candidate is `.local-mac/epaperprovision-v3`; firmware is
`build-usb-v3/epaper_receiver.bin`. Do not overwrite or erase the known-good
`build-release` artifact or stored credentials. This candidate includes the
separately tested idle UART byte fix; it is not an isolated UART experiment.

On 2026-09-21, 18 native tests, the complete remote-epaper Go suite, and
Go-to-native v2/v3 provisioning round trips passed. Scoped Go lint reported
zero issues. Fault tests cover valid stale replies before a fresh reply,
fragmented/prefixed replies, wrong operations, exhausted/no-progress reads,
write failures, and prevention of repeated erase after ambiguous failure.
No physical flash or visible-screen acceptance is implied by these results.
