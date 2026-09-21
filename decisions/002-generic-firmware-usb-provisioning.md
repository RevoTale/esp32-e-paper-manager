# ADR-002: Use generic firmware with USB-only provisioning

## Status

Accepted.

## Date

2026-09-02.

## Context

Device secrets must be installed exclusively through physical USB. Building a
different UF2 for every Pico would place credentials inside build inputs,
caches, artifacts, and operator workflows and would require reflashing merely
to rotate a key.

## Decision

Build one reproducible generic UF2 with no device or Wi-Fi secrets. After
flashing, a trusted host tool provisions the Pico over USB.

The USB-only record contains the Wi-Fi authentication mode, SSID, passphrase,
manager endpoint, device identity, device key, and IANA timezone. Wi-Fi
credentials and timezone can be changed without rebuilding or reflashing
firmware. Firmware must not accept initial Wi-Fi setup or credential changes
over Wi-Fi or the public manager API.

The host tool generates a 256-bit device key using the host operating system's
cryptographic RNG and atomically installs matching identities on Pico and the
home manager through local privileged interfaces. It must not print the key,
place it in process arguments, send it through the public API, or retain an
unnecessary plaintext export.

Until every required field is present and the complete record is
integrity-valid, firmware keeps Wi-Fi and network updates disabled. USB
diagnostics and reprovisioning remain available. Provisioning data is versioned
and supports atomic replacement, interrupted write recovery, rotation, and
factory reset. A failed rotation preserves the previous valid record or leaves
the device in an explicit unprovisioned USB-only state.

## Alternatives considered

### Device-specific UF2

Rejected. It spreads secrets into build artifacts and makes rotation depend on
firmware rebuild/reflash.

### Provisioning over Wi-Fi

Rejected. An unprovisioned device has no established remote identity or trusted
encrypted channel.

### Encrypt Pico flash configuration with a key stored in the same flash

Rejected as a false security boundary. It does not protect against physical
flash extraction. Physical compromise is addressed through device security,
secret rotation, and manager-side revocation.

## Consequences

- One UF2 can be built, tested, and distributed without secrets.
- First use requires USB and a trusted provisioning host.
- Device-key and Wi-Fi credential rotation do not require replacing firmware.
- The host tool and manager need a local privileged provisioning interface and
  strict secret-storage permissions.
- Power loss during provisioning must leave either the previous valid record or
  a clearly unprovisioned state, never a partially trusted configuration.
