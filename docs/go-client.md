# Go client library

Public Go packages share the firmware's release tag. In your own module:

```sh
go get github.com/RevoTale/esp32-e-paper-manager@v0.1.0
```

Choose the actual version from GitHub Releases. There is no separate client
repository/version. The module currently requires Go 1.26.

- screenclient: EPS2 binding, capabilities, full/partial transfers and recovery.
- securetransport: encrypted authenticated Wi-Fi using the USB-provisioned key.
- screenwire: typed records and diagnostics.
- display: monochrome frames; engine: bounded HTML/inline-CSS rendering.
- screendelivery: delivery policy and region planning.

Open the provisioned ESP32 network endpoint with a deadline, then bind EPS2:

```go
var nonce securetransport.ClientNonce
if _, err := rand.Read(nonce[:]); err != nil {
    return err
}
stream, err := securetransport.HostHandshake(connection, key, deviceID, nonce)
if err != nil {
    return err
}
client, err := screenclient.New(rand.Reader)
if err != nil {
    return err
}
caps, err := client.Bind(stream)
if err != nil {
    return err
}
frame, err := display.NewFrame(
    display.Size{Width: int(caps.Width), Height: int(caps.Height)},
    int(caps.Stride), pixels,
)
if err != nil {
    return err
}
return client.Send(frame)
```

Import crypto/rand and this module's screenclient/securetransport/display packages.
Connection, key/device identity and rendered pixels come from your application.
Retain the client across reconnects; rebind and reconcile pending transactions
before sending another frame. Do not blindly replay an uncertain physical
update. The firmware advertises and enforces its refresh cadence and ownership.

For HTML producers, run epaper-manager and use the authenticated
[screen API](screen-api.md). The manager owns rendering, batching, pixel damage,
refresh scheduling and reconnection. Direct screenclient consumers send pixels.

For USB see [the host guide](eps2-usb-host.md). The ready epaperscreen command
supervises its serial transport.
