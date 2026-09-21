# Capability Map: Remote e-paper

Status: approved on 2026-08-30.

| Module id | Responsibility | Depends on |
|---|---|---|
| `panel-driver` | GDEY075T7 power, SPI commands, full refresh, BUSY handling, and deep sleep | — |
| `frame-protocol` | Versioned 1-bit frame envelope, chunks, CRC, acknowledgements, and errors | — |
| `device-runtime` | Frame ownership, display lifecycle, bounded queueing, and USB-over-Wi-Fi arbitration | `panel-driver`, `frame-protocol` |
| `usb-transport` | Streaming frame protocol over TinyGo USB CDC | `frame-protocol`, `device-runtime` |
| `wifi-transport` | The same frame protocol over CYW43439 TCP | `frame-protocol`, `device-runtime` |
| `host-client` | Go CLI image conversion and USB/TCP transfer | `frame-protocol` |
| `terminal-renderer` | Host-side text and terminal snapshot rendering | `host-client` |

Build order:

1. `panel-driver` and `frame-protocol` foundations.
2. `device-runtime` with `usb-transport` as the first end-to-end slice.
3. `host-client` image transfer over USB.
4. `wifi-transport` using the same protocol and USB-priority rules.
5. `terminal-renderer` and operational polish.

Cross-cutting requirements:

- Correctness, energy use, memory, CPU wake time, transfer time, and radio use
  are evaluated for every material decision.
- Primary documentation and target-version source take precedence over similar
  drivers and recollection.
- External input is validated at the transport/protocol boundary.
- No module may allocate an additional full-frame copy on the Pico without a
  measured and documented reason.
- Partial and fast refresh remain experimental until separately verified on
  the physical GDEY075T7.
