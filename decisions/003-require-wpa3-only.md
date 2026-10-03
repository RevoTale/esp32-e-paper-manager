# ADR-003: Require WPA3-only Wi-Fi

## Status

Accepted.

## Date

2026-09-02.

## Context

Pico 2 W is expected to carry private dashboard content over home Wi-Fi. The
hardware and pinned TinyGo Wi-Fi driver expose WPA3 support, but target-router
interoperability still requires a physical test. Supporting older modes would
increase configuration, testing, and downgrade paths while weakening the
requested security baseline.

## Decision

Join only WPA3-SAE networks. Forbid WPA2, WPA1, TKIP, open Wi-Fi, WPA2/WPA3
transition mode, and automatic downgrade. Provisioning must reject every other
authentication mode.

If WPA3 association fails, report a typed, visible diagnostic and retain USB
access. Do not retry with a weaker mode. Firmware acceptance requires physical
join, reconnect, power-cycle, and failure-diagnostic tests against the target
WPA3 access point using the pinned TinyGo driver.

Application-layer mutual authentication and AES-256-GCM remain mandatory;
WPA3 protects the radio link but does not replace the device protocol.

## Alternatives considered

### WPA3 preferred with explicit WPA2-AES fallback

Rejected. It improves compatibility but violates the chosen WPA3-only security
policy and expands downgrade-sensitive behavior.

### WPA2/WPA3 transition mode

Rejected. The device must prove a WPA3-only association rather than rely on a
mixed-mode access point.

## Consequences

- Routers or TinyGo driver versions that cannot complete WPA3-SAE are
  incompatible until corrected or replaced.
- Wi-Fi cannot be accepted from documentation or compile success alone; it
  needs physical WPA3 tests on the target hardware and access point.
- Failure leaves the device manageable through USB without weakening security.
