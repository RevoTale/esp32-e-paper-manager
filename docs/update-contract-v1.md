# Update envelope v1

The `update` package is the transport-neutral contract shared by USB, the home
manager, and the encrypted device link. It does not import the parser,
renderer, runtime, or panel.

## Ownership and limits

- The HTML profile is `document.Version1` and `document.ProfileDashboard`.
- HTML is valid UTF-8, non-empty, and at most 32,768 bytes.
- `document.Source` and decoded `update.Request` borrow their input bytes. The
  caller must retain immutable storage until processing completes.
- The update ID is 16 non-zero bytes. Its content identity is SHA-256 over the
  exact HTML bytes.
- Display time is exactly `YYYY-MM-DD HH:MM`. The timezone is a separately
  provisioned IANA name of at most 64 bytes; the host must resolve it and make
  it match the provisioned device value.
- Repeating the same ID, UTC instant, timestamp, timezone, document profile,
  version, and content hash is the same intent.

## Binary layout

All integers are big-endian. The fixed header is 160 bytes and the trailer is
four bytes.

| Offset | Size | Field |
|---:|---:|---|
| 0 | 4 | ASCII `EPU2` |
| 4 | 1 | envelope version `1` |
| 5 | 1 | document profile |
| 6 | 1 | document version |
| 7 | 1 | reserved zero |
| 8 | 4 | HTML byte length |
| 12 | 8 | positive Unix UTC seconds |
| 20 | 1 | display-time length `16` |
| 21 | 1 | timezone length `1..64` |
| 22 | 2 | reserved zero |
| 24 | 16 | update ID |
| 40 | 32 | SHA-256 HTML identity |
| 72 | 16 | ASCII display time |
| 88 | 4 | reserved zero |
| 92 | 64 | timezone, then canonical zero padding |
| 156 | 4 | IEEE CRC-32 of bytes `0..155` |
| 160 | N | HTML payload |
| 160+N | 4 | IEEE CRC-32 of header and payload |

The checked-in `testdata/update-v1-request.hex` vector freezes this layout.
Unknown versions, profiles, non-canonical padding, invalid metadata,
truncation, trailing bytes, CRC mismatches, and content-hash mismatches fail
closed.

## Diagnostics

Failures identify one stable stage (`receive` through `refresh`) and code. A
bounded limit diagnostic may add a lowercase identifier, observed value,
maximum, and node index. Successful results must contain ID and content hash;
rejected results must contain a valid diagnostic.

CRC-32 detects accidental corruption only. It does not authenticate a sender.
USB relies on physical access; the network link must add the separately
specified HMAC-SHA-256 authentication and AES-256-GCM record protection.

## Sources

- Go SHA-256: https://pkg.go.dev/crypto/sha256
- Go IEEE CRC-32: https://pkg.go.dev/hash/crc32
- Go big-endian encoding: https://pkg.go.dev/encoding/binary
- Go UTF-8 validation: https://pkg.go.dev/unicode/utf8
- TinyGo standard-library support: https://tinygo.org/docs/reference/lang-support/stdlib/
