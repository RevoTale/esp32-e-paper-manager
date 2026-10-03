# V2 acceptance evidence

## Automated evidence

- `scripts/quality.sh task`: lint 0, changed executable-line coverage 90.0%,
  total coverage 88.2% against a 75.0% baseline, USB and WPA3 TinyGo builds.
- `scripts/quality.sh full`: PASS twice in 70 and 54 seconds with vet, race,
  module verification, and protocol/HTML/render resource probes.
- Host tests cover parser/CSS/layout/raster goldens, overflow, USB fragmentation
  and detach, atomic/torn flash, provisioning redaction, HTTPS auth and limits,
  restart persistence, encrypted link tamper/wrong-key/replay behavior,
  two-phase accepted/terminal status, USB arbitration, and backoff.
- Network tests cover malformed endpoints, capped retry, USB priority, initial
  join retry, and LINK/DEAUTH recovery order: WPA3 rejoin, DHCP, then DNS.
- New files obey 300 lines/file, 60 lines/function, and complexity <=10.
  Pre-existing oversized debt remains reported and did not grow.

## Physical evidence still required

- 20 consecutive USB HTML updates including detach/retry/cold boot.
- WPA3-only association; explicit rejection of WPA2/open networks.
- DNS/TCP reconnect, router loss/restart, USB priority, provisioning rotation,
  repeated power cycles, and manager terminal status after physical refresh.
- Visible timestamp rectangle and overflow behavior on the exact 800x480 panel.
- BUSY timing, >=180-second refresh spacing, current draw/radio-on time, and
  battery-life measurements.
- Dual-stack target acceptance is blocked by the Pico IP-stack limitation
  documented in `security-network.md`.

Automated compilation is not physical acceptance. Do not mark V2-16, V2-17,
or final V2-18 complete until the corresponding hardware evidence is recorded.
