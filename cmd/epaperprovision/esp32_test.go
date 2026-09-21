package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
	"go.bug.st/serial"
)

func TestESP32InspectDoesNotAssertDTRAndRequiresExplicitPort(t *testing.T) {
	original := openSerial
	defer func() { openSerial = original }()
	device := &fakeSerial{}
	opened := 0
	openSerial = func(name string, mode *serial.Mode) (serialDevice, error) {
		opened++
		if name != "test-port" || mode.InitialStatusBits == nil || mode.InitialStatusBits.DTR || mode.InitialStatusBits.RTS {
			t.Fatal("ESP32 requires explicit modem output bits")
		}
		return device, nil
	}
	var out bytes.Buffer
	if err := run([]string{"-target", "esp32", "-port", "test-port", "inspect"}, nil, &out); err != nil {
		t.Fatal(err)
	}
	if device.dtr || !device.closed {
		t.Fatal("unexpected DTR or leaked port")
	}
	if opened != 1 {
		t.Fatal("unexpected serial open count")
	}
}

func TestESP32RequiresExplicitPortBeforeAnySerialAccess(t *testing.T) {
	original := openSerial
	t.Cleanup(func() { openSerial = original })
	openSerial = func(string, *serial.Mode) (serialDevice, error) {
		t.Fatal("invalid target or autodiscovery accessed serial")
		return nil, nil
	}
	for _, target := range []string{"esp32", "unknown"} {
		var out bytes.Buffer
		if err := run([]string{"-target", target, "inspect"}, nil, &out); err == nil {
			t.Fatal("target without an explicit supported port accepted")
		}
	}
}

func TestESP32BuildRequestCreatesExplicitWPA2(t *testing.T) {
	codec := provision.ESP32Codec()
	if codec.AuthMode() != provision.AuthWPA2PSK {
		t.Fatal("policy")
	}
	_, err := boardCodec("esp32")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := boardCodec(strings.Repeat("x", 10)); err == nil {
		t.Fatal("unknown policy")
	}
}
