package screenusbhost

import (
	"errors"
	"io"
	"strings"
	"time"

	"go.bug.st/serial"
)

// ESP32 serial endpoints use esp32:/absolute/serial/path. Pico paths are unchanged.
// UART bridges do not expose CDC connection lifetime, so their owner retires
// idle bindings; the parent explicitly rebinds the same claim before sending.
func esp32Endpoint(name string) bool { return strings.HasPrefix(name, "esp32:") }

var openESP32Port = func(name string) (serialPort, error) {
	return serial.Open(name, &serial.Mode{BaudRate: 115200, InitialStatusBits: &serial.ModemOutputBits{}})
}

func esp32Proxy(name string, input io.Reader, output io.Writer) (result error) {
	name = strings.TrimPrefix(name, "esp32:")
	if name == "" {
		return ErrConfiguration
	}
	p, err := openESP32Port(name)
	if err != nil {
		return proxyFailure{20, ErrWorker}
	}
	defer func() { result = errors.Join(result, p.Close()) }()
	if err := p.SetReadTimeout(180 * time.Second); err != nil {
		return proxyFailure{21, ErrWorker}
	}
	// No deliberate DTR/RTS toggles. Unix drivers may still pulse lines on open;
	// cold setup and abort-failure recovery require physical power removal.
	return serveProxy(input, output, p)
}
