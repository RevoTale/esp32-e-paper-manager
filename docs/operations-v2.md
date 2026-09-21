# V2 build and operation

Run in the existing Dev Container:

```sh
cd /workspaces/pico-sandbox/experiments/03-remote-epaper
./scripts/quality.sh task
./scripts/quality.sh full
mkdir -p dist
tinygo build -target=pico2-w -scheduler=tasks -o dist/remote-epaper-usb.uf2 ./cmd/device
tinygo build -target=pico2-w -scheduler=tasks -tags=wifi -o dist/remote-epaper-wifi.uf2 ./cmd/device
go build -trimpath -o dist/epaperhtml ./cmd/epaperhtml
go build -trimpath -o dist/epaperprovision ./cmd/epaperprovision
go build -trimpath -o dist/epaper-manager ./cmd/epaper-manager
```

Flash the generic Wi-Fi UF2 through direct macOS `/Volumes/RP2350` copying as
already agreed. Provision afterward over USB; do not build personalized UF2s.

Коли на Pico вже працює TinyGo CDC firmware, фізичний BOOTSEL зазвичай не
потрібен. На macOS переведи CDC у bootloader через 1200 baud:

```sh
stty -f /dev/cu.usbmodemNNN 1200
```

Після появи `/Volumes/RP2350` скопіюй UF2 напряму. Це перевірено на Pico 2 W:
TinyGo викликає bootloader, коли CDC отримує 1200 baud із вимкненим DTR.
Фізичний BOOTSEL лишається fallback, якщо CDC не з'явився або поточна firmware
не підтримує цей reset. Не повертайся до OrbStack USB passthrough без окремої
причини.

For the USB-only physical smoke test, build the macOS client in the container,
put Pico into BOOTSEL, then run on macOS:

```sh
./scripts/flash-usb-macos.sh /dev/cu.usbmodem1101
```

It copies the generic UF2, waits without a hidden overall timeout for the mass
storage volume to disappear and the selected CDC path to return, performs the
bounded diagnostic handshake, and submits `testdata/post-flash-success.html`.
The visible page proves the full flash -> boot -> CDC -> protocol -> render ->
panel path. A successful handshake alone does not prove a physical refresh.

USB HTML remains the default:

```sh
./dist/epaperhtml dashboard.html
```

Remote submission uses the manager, not Pico:

```sh
./dist/epaperhtml -manager https://manager.example \
  -token-file ./manager-api-token -device-id DEVICE_ID dashboard.html
```

Manager startup requires a TLS certificate/key, private API-token file,
private enrollment file, and restart-safe state file:

```sh
./dist/epaper-manager -tls-cert cert.pem -tls-key key.pem \
  -token-file manager-api-token -enrollment device-enrollment.json \
  -state manager-state.json
```

The API token and enrollment/state files must be mode `0600`. The state file
may contain pending HTML and is therefore sensitive even though it contains no
device key.
