# Bounded EPS2 compression

2026-09-07. Software-qualified optional encoding; not partial refresh and not a
measurement of electrical energy, latency or peak MCU memory.

## Contract

Capabilities require the existing raw-mono1/full-refresh bits (0 and 1).
Optional bit 2 (`FeaturePackBits`) admits `DataPacked`, operation 11. Existing
operation numbers, status lengths, physical profile and raw `Data` are unchanged.
Unknown feature bits still reject. A raw-only peer receives only raw packets.

`DataPacked` has the usual epoch/transaction/pass and **decoded** offset.
Its payload is a little-endian u16 decoded length followed by byte-run controls.
Decoded length is 1..negotiated MaxChunk, at most 1024. Encoded payload is 4..1024.
Controls 0..127 introduce 1..128 literal bytes; 129..255 repeat the next byte
128..2 times. Control 128 is forbidden. All input must be consumed and exactly
the announced output produced; truncation, extra bytes and over-expansion reject.

This deliberately restricted chunk codec reuses the established
[TIFF 6.0 section 9 PackBits controls](https://www.itu.int/itudoc/itu-t/com16/tiff-fx/docs/tiff6.pdf).
It is **not a TIFF container or TIFF row codec**: chunks may span logical rows,
and TIFF's no-op is excluded. No dictionary, cross-chunk state, remote cache,
allocation from wire lengths or native library is involved.

The manager uses a deterministic greedy encoder only when prefix plus encoded
bytes are strictly smaller than raw input. This guarantees no larger data record
and unchanged ACK count, not globally optimal compression. Mixed raw/packed
chunks are valid; both planes and SHA-256 cover identical canonical decoded bytes.

## Receiver safety and ownership

Authenticate an entire network record first, then check binding, transaction,
capability and bounds. One serialized device owner shares a 1024-byte decode
scratch. Decode completely before calling the existing session receiver; it
still validates pass, exact offset, row padding, digest and completeness before
Commit. Never stream partially decoded output directly to SPI.

A malformed packed block aborts its matching incomplete transaction, including
Ready state. It cannot leave rejected staging eligible for a later Commit.
An unrelated/stale transaction cannot use decoder failure to abort the current
owner. Sink cleanup errors remain attached to the failure. No physical sequence,
plane polarity, BUSY condition, full-refresh cadence or power policy changes.

The scratch adds 1024 bytes to the Device allocation. TinyGo static-RAM output
alone does not measure that heap cost. Firmware/resource gates must account for
this explicitly. Encoded ciphertext lengths reveal per-chunk compressibility;
AEAD hides content and authenticates records but does not hide traffic shape.

## Measured software evidence

Fresh race tests and strict lint pass for codec/wire/link/client/integration.
Standalone codec coverage is 100%; a five-second fuzz run completed 768,053
executions. Independent literal-control fixtures cover run bounds, malformed
lengths, overflow, no-op rejection and exact consumption. Roundtrip allocation
test reports zero codec allocations with caller-supplied buffers.

Actual complete EPS2 exchanges, 800×480, two 48,000-byte planes, 1000-byte chunks;
includes Hello/Acquire/Bind/Begin/Commit and every data reply:

| Synthetic target | Raw bytes | Negotiated packed bytes |
| --- | ---: | ---: |
| Solid white | 107480 | 13208 |
| Non-repeating bytes | 107480 | 107480 |
| Alternating raw/solid chunks | 107480 | 60344 |

A real native HTML fixture submitted through HTTPS and sent through actual
handshake + AEAD + EPS2 produces **114072 raw versus 22334 compressed bytes**.
Both traces match the accepted buffered driver's complete SPI/GPIO/delay trace.
This uses recording hardware, not the physical panel. USB/Ethernet/TCP/WLAN
link framing is outside these application-stream counters; retransmission and
real radio/powered-wait energy remain target acceptance work.
