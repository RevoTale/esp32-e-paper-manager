package main

import (
	"errors"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
	"github.com/RevoTale/esp32-e-paper-manager/provisionclient"
	"go.bug.st/serial"
)

func provisioningClient(port serialDevice, codec provision.Codec, correlated bool) (*provisionclient.Client, error) {
	if !correlated {
		return provisionclient.NewFor(port, codec)
	}
	if err := port.SetReadTimeout(500 * time.Millisecond); err != nil {
		return nil, err
	}
	return provisionclient.NewCorrelated(port)
}

func boardCodec(target string) (provision.Codec, error) {
	switch target {
	case "pico":
		return provision.Codec{}, nil
	case "esp32":
		return provision.ESP32Codec(), nil
	default:
		return provision.Codec{}, errors.New("target must be pico or esp32")
	}
}

func connectFor(requested string, codec provision.Codec) (serialDevice, func(), error) {
	if codec.AuthMode() == provision.AuthWPA3SAE {
		return connect(requested)
	}
	if requested == "" || requested == "auto" {
		return nil, func() {}, errors.New("ESP32 requires an explicit -port")
	}
	// Do not deliberately pulse EN/GPIO0. The OS may still pulse modem lines
	// during open: first access must follow a safe, fully powered-off setup.
	mode := &serial.Mode{BaudRate: 115200, InitialStatusBits: &serial.ModemOutputBits{}}
	port, err := openSerial(requested, mode)
	if err != nil {
		return nil, func() {}, err
	}
	closePort := func() { _ = port.Close() }
	if err := port.SetReadTimeout(20 * time.Second); err != nil {
		closePort()
		return nil, func() {}, err
	}
	return port, closePort, nil
}
