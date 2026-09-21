package main

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
	"go.bug.st/serial"
)

type correlatedSerial struct {
	fakeSerial
	timeout time.Duration
}

func (d *correlatedSerial) SetReadTimeout(timeout time.Duration) error {
	d.timeout = timeout
	return nil
}

func (d *correlatedSerial) Write(request []byte) (int, error) {
	if request[4] != 3 {
		return 0, io.ErrUnexpectedEOF
	}
	var reply [512]byte
	err := provision.ESP32Codec().EncodeResponse(reply[:], provision.Response{
		Operation: provision.OperationInspect, State: provision.StateBlank,
	})
	if err != nil {
		return 0, err
	}
	reply[4] = 3
	copy(reply[392:408], request[486:502])
	binary.BigEndian.PutUint32(reply[508:], crc32.ChecksumIEEE(reply[:508]))
	d.reply.Reset(append(make([]byte, 256), reply[:]...))
	return len(request), nil
}

func TestRunCorrelatedESP32Inspect(t *testing.T) {
	original := openSerial
	t.Cleanup(func() { openSerial = original })
	device := &correlatedSerial{}
	openSerial = func(string, *serial.Mode) (serialDevice, error) { return device, nil }
	err := run([]string{"-target", "esp32", "-usb-v3", "-port", "test", "inspect"}, nil, io.Discard)
	if err != nil || device.timeout != 500*time.Millisecond || !device.closed {
		t.Fatal(err, device.timeout, device.closed)
	}
}

func TestRunRejectsCorrelatedPicoBeforeOpeningPort(t *testing.T) {
	original := openSerial
	t.Cleanup(func() { openSerial = original })
	openSerial = func(string, *serial.Mode) (serialDevice, error) {
		t.Fatal("unsupported protocol opened port")
		return nil, nil
	}
	if err := run([]string{"-usb-v3", "-port", "test", "inspect"}, nil, io.Discard); err == nil {
		t.Fatal("Pico v3 accepted")
	}
}
