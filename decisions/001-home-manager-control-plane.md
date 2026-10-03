# ADR-001: Use a home manager as the public control plane

## Status

Accepted.

## Date

2026-09-02.

## Context

The display must be securely manageable by an authorized user connecting from
any Internet source address. Pico 2 W must still receive and render the bounded
HTML profile, USB must remain the preferred path, and credentials must be
installed only through physical USB.

Exposing a plaintext HTTP bearer-token endpoint directly on Pico would reveal
the credential and HTML to a network observer. Direct Internet exposure would
also make the resource-constrained device responsible for public TLS, scanning,
authentication floods, and denial-of-service handling.

## Decision

Use a trusted home server as the central manager:

```text
remote user -> HTTPS or VPN -> home manager
home manager -> mutually authenticated encrypted Wi-Fi link -> Pico
USB host -> preferred local provisioning/update path -> Pico
```

The manager is the only public control plane. Its default deployment is the
home server reached through an outbound tunnel, but the provisioned endpoint
may later move to a public server. Pico is not port-forwarded and does not
require a public IP address. Pico initiates or polls the device link, receives a
bounded pending HTML update, validates and renders it locally, reports a bounded
result, and then follows the e-paper power lifecycle.

The `device-link` is location-independent: DNS, IPv4, and IPv6 endpoints are
allowed after their respective acceptance tests. It must not assume a private
address, same subnet, broadcast discovery, or trust based on source IP.

WPA3 protects the radio link. Independently, a USB-provisioned 256-bit device
key authenticates both manager and Pico through HMAC-SHA-256 challenge-response;
domain-separated session keys and AES-256-GCM protect records with strict
sequence/replay validation. The long-term key is never sent over Wi-Fi.

Public user credentials and device keys are separate. The public API must not
return, log, accept, or derive the Pico device key from a user credential.

## Alternatives considered

### Direct public HTTP endpoint on Pico

Rejected. Plain HTTP exposes secrets and content, while direct public exposure
creates unnecessary denial-of-service and patching risk on the microcontroller.

### Public cloud relay with an outbound Pico connection

Viable if the home manager cannot be made securely reachable, especially under
CGNAT. Not selected because the user chose a central server in the home network.
The `device-link` contract should remain reusable if this decision changes.

### VPN endpoint directly on Pico

Rejected for the first implementation. It adds substantial firmware, RAM,
cryptographic, interoperability, and energy cost while a home gateway/server can
provide the same public boundary more safely.

## Consequences

- The manager needs HTTPS/VPN exposure, authentication, rate limits, updates,
  monitoring, and a bounded retention policy. A home deployment may obtain
  public reachability through an outbound tunnel without opening the router's
  inbound firewall.
- A network observer cannot recover the device key or HTML and cannot forge or
  replay an accepted update under the defined cryptographic assumptions.
- Compromise of the manager, Pico firmware/flash, or a trusted provisioning host
  remains in scope for key rotation and incident recovery; encryption cannot
  make these trusted endpoints harmless.
- The project now contains both firmware and a small home-manager service, with
  a shared versioned update contract and independent tests.
- USB remains operational when the manager or Wi-Fi is unavailable and preempts
  incomplete network work.
