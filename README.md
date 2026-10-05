# ESP32 e-paper manager

Render a bounded HTML/inline-CSS dashboard in Go and deliver it to an ESP32
Waveshare e-paper board over an authenticated, encrypted Wi-Fi link or USB.

- Go rendering and management; C/ESP-IDF firmware for the ESP32 receiver.
- Images, text and layout resolved on the server; bounded device-side transfers.
- Recovery-aware delivery and configurable normal/urgent full-refresh policy.
- USB provisioning keeps device credentials out of dashboard requests.

## Getting started

Pushes to main publish versioned firmware and manager images after CI passes.
The [Go client library](docs/go-client.md) uses the same version tag.
See [automatic releases](docs/releasing.md) and
[firmware update instructions](docs/firmware-release.md).

1. Open the repository in its Dev Container.
2. Run `make quality` to test Go, native C, Go/C interoperability and TinyGo
   compatibility, then build the ESP32 firmware and Linux manager.
3. Follow the [receiver guide](firmware/esp32/README.md) for hardware and USB
   provisioning. Do not overwrite an existing enrollment or flash partition.
4. Follow [container deployment](docs/container-deployment.md) and the
   [screen API guide](docs/screen-api.md) to submit a dashboard.

The current HTML API is for trusted producers and is loopback-only. E-paper
refresh completion in logs is not a substitute for visually checking a frame.

See [documentation](docs/README.md) for supported properties, protocol,
security boundaries, source references and known limitations.
