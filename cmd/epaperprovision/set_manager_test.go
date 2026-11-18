package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
	"go.bug.st/serial"
)

type endpointPort struct {
	fakeSerial
	failTimeout bool
	unsupported bool
	writes      int
}

func (p *endpointPort) SetReadTimeout(d time.Duration) error {
	if d == 500*time.Millisecond && p.failTimeout {
		return io.ErrClosedPipe
	}
	return nil
}

func (p *endpointPort) Write(request []byte) (int, error) {
	p.writes++
	if request[5] != 7 || request[4] != 3 || p.unsupported {
		return 0, errors.New("unknown operation")
	}
	var reply [512]byte
	err := provision.ESP32Codec().EncodeResponse(reply[:], provision.Response{Operation: provision.OperationInspect,
		State: provision.StateProvisioned, Auth: provision.AuthWPA2PSK, DeviceID: [16]byte{3}, Generation: 2,
		SSID: "example", Timezone: "Europe/Kyiv", Manager: "tcp://paper.home.arpa:9757"})
	if err != nil {
		return 0, err
	}
	reply[4], reply[5] = 3, 7
	copy(reply[392:], request[486:502])
	binary.BigEndian.PutUint32(reply[508:], crc32.ChecksumIEEE(reply[:508]))
	p.reply.Reset(reply[:])
	return len(request), nil
}

func TestSetManagerCLI(t *testing.T) {
	original := openSerial
	t.Cleanup(func() { openSerial = original })
	for _, tc := range []struct {
		args        []string
		failTimeout bool
		unsupported bool
		ok          bool
		writes      int
	}{
		{[]string{"-target", "pico", "set-manager", "paper"}, false, false, false, 0},
		{[]string{"-target", "esp32", "set-manager"}, false, false, false, 0},
		{[]string{"-target", "esp32", "set-manager", "tcp://paper:0"}, false, false, false, 0},
		{[]string{"-target", "esp32", "-port", "test", "set-manager", "192.168.1.999"}, false, false, false, 0},
		{[]string{"-target", "esp32", "set-manager", "paper"}, false, false, false, 0},
		{[]string{"-target", "esp32", "-port", "test", "set-manager", "paper"}, true, false, false, 0},
		{[]string{"-target", "esp32", "-port", "test", "set-manager", "paper"}, false, true, false, 1},
		{[]string{"-target", "esp32", "-port", "test", "set-manager", "paper.home.arpa"}, false, false, true, 1},
	} {
		port := &endpointPort{failTimeout: tc.failTimeout, unsupported: tc.unsupported}
		openSerial = func(string, *serial.Mode) (serialDevice, error) { return port, nil }
		var out bytes.Buffer
		err := run(tc.args, nil, &out)
		if (err == nil) != tc.ok || port.writes != tc.writes {
			t.Fatal(tc.args, err, port.writes)
		}
		if tc.ok && out.String() != "manager=\"tcp://paper.home.arpa:9757\" generation=2 changed=false\n" {
			t.Fatal(out.String())
		}
	}
}
