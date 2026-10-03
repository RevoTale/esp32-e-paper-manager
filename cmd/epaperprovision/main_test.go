package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
	"go.bug.st/serial"
)

type fakeSerial struct {
	request   bytes.Buffer
	reply     bytes.Reader
	closed    bool
	dtr       bool
	state     provision.State
	code      provision.Code
	config    provision.Config
	mutate    func(*provision.Response)
	loseACK   bool
	writeFail bool
	observe   func(provision.Request)
}

func (device *fakeSerial) Write(value []byte) (int, error) {
	count, err := device.request.Write(value)
	if device.request.Len() == provision.RequestSize && device.reply.Len() == 0 {
		request, decodeErr := provision.DecodeRequest(device.request.Bytes())
		if decodeErr != nil {
			return count, decodeErr
		}
		if device.observe != nil {
			device.observe(request)
		}
		if device.writeFail && request.Operation == provision.OperationRotate {
			return count / 2, io.ErrClosedPipe
		}
		wire := make([]byte, provision.ResponseSize)
		if err := provision.EncodeResponse(wire, device.response(request)); err != nil {
			return count, err
		}
		device.reply.Reset(wire)
		if device.loseACK && request.Operation == provision.OperationRotate {
			device.reply.Reset(nil)
		}
		device.request.Reset()
	}
	return count, err
}

func (device *fakeSerial) response(request provision.Request) provision.Response {
	if request.Operation == provision.OperationErase {
		device.state = provision.StateBlank
	}
	if request.Operation == provision.OperationProvision || request.Operation == provision.OperationRotate {
		device.state = provision.StateProvisioned
		device.config = request.Config
	}
	r := provision.Response{Operation: request.Operation, State: device.state, Code: device.code}
	if r.State == provision.StateProvisioned {
		if device.config.DeviceID == ([16]byte{}) {
			device.config = provision.Config{Auth: provision.AuthWPA3SAE, DeviceID: [16]byte{1},
				SSID: "fixture", Manager: "tcp://manager.example:9757", Timezone: "Europe/Kyiv"}
		}
		r.Generation, r.Auth, r.DeviceID = 1, device.config.Auth, device.config.DeviceID
		r.SSID, r.Manager, r.Timezone = device.config.SSID, device.config.Manager, device.config.Timezone
	}
	if device.mutate != nil {
		device.mutate(&r)
	}
	return r
}

func TestRunPrintsAuthoritativeStateForAcknowledgedFailure(t *testing.T) {
	original := openSerial
	defer func() { openSerial = original }()
	device := &fakeSerial{state: provision.StateUnknown, code: provision.CodeStorage}
	openSerial = func(string, *serial.Mode) (serialDevice, error) { return device, nil }
	var out bytes.Buffer
	err := run([]string{"-port", "test", "inspect"}, strings.NewReader(""), &out)
	if err == nil || !strings.Contains(out.String(), "code=3 state=3") {
		t.Fatal(out.String(), err)
	}
}

func TestRunRotateInspectsIdentityAndWritesPrivateEnrollment(t *testing.T) {
	original := openSerial
	defer func() { openSerial = original }()
	device := &fakeSerial{state: provision.StateProvisioned}
	openSerial = func(string, *serial.Mode) (serialDevice, error) { return device, nil }
	registry := filepath.Join(t.TempDir(), "device.json")
	input := strings.NewReader(`{"ssid":"secure","passphrase":"long-secret","manager":"tcp://manager.example:9757"}`)
	if err := run([]string{"-port", "test", "-registry", registry, "rotate"}, input, io.Discard); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(registry)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%v err=%v", info.Mode().Perm(), err)
	}
}

func TestRunRejectsUsageAndUnprovisionedRotation(t *testing.T) {
	if err := run(nil, strings.NewReader(""), io.Discard); err == nil {
		t.Fatal("missing operation accepted")
	}
	original := openSerial
	defer func() { openSerial = original }()
	openSerial = func(string, *serial.Mode) (serialDevice, error) { return &fakeSerial{}, nil }
	err := run([]string{"-port", "test", "-registry", filepath.Join(t.TempDir(), "device.json"), "rotate"},
		strings.NewReader(`{"ssid":"secure","passphrase":"long-secret","manager":"tcp://manager.example:9757"}`), io.Discard)
	if err == nil {
		t.Fatal("unprovisioned rotation accepted")
	}
}

func (device *fakeSerial) Read(value []byte) (int, error)     { return device.reply.Read(value) }
func (device *fakeSerial) SetReadTimeout(time.Duration) error { return nil }
func (device *fakeSerial) SetDTR(value bool) error            { device.dtr = value; return nil }
func (device *fakeSerial) Close() error                       { device.closed = true; return nil }

func TestRunInspectAndEraseConfirmation(t *testing.T) {
	original := openSerial
	defer func() { openSerial = original }()
	device := &fakeSerial{}
	openSerial = func(string, *serial.Mode) (serialDevice, error) { return device, nil }
	var output bytes.Buffer
	if err := run([]string{"-port", "test", "inspect"}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "state=0") || !device.closed || device.dtr {
		t.Fatalf("output=%q closed=%v dtr=%v", output.String(), device.closed, device.dtr)
	}
	if _, err := parseOperation("erase", false); err == nil {
		t.Fatal("unconfirmed erase accepted")
	}
	if operation, err := parseOperation("erase", true); err != nil || operation != provision.OperationErase {
		t.Fatalf("operation=%d err=%v", operation, err)
	}
}

func TestBuildProvisionRequestCreatesPrivatePendingEnrollment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "device.json")
	input := strings.NewReader(`{"ssid":"secure","passphrase":"long-secret","manager":"tcp://manager.example:9757"}`)
	request, pending, err := buildRequest(nil, provision.OperationProvision, path, input)
	if err != nil || request.Config.Validate() != nil {
		t.Fatalf("request=%+v err=%v", request, err)
	}
	defer func() { _ = pending.Close() }()
	info, err := os.Stat(path + ".pending")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%v", info.Mode().Perm())
	}
	if _, err = os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("active enrollment exists before ACK: %v", err)
	}
	if _, _, err = buildRequest(nil, provision.OperationProvision, "", input); err == nil {
		t.Fatal("missing registry accepted")
	}
}

func TestOperationAndClientValidation(t *testing.T) {
	for _, value := range []string{"inspect", "provision", "rotate", "diagnose"} {
		if _, err := parseOperation(value, false); err != nil {
			t.Fatalf("%s: %v", value, err)
		}
	}
	if _, err := parseOperation("unknown", false); err == nil {
		t.Fatal("unknown operation accepted")
	}
	if _, _, err := buildRequest(nil, provision.OperationProvision, "registry", io.LimitReader(strings.NewReader("{}"), 10)); err == nil {
		t.Fatal("invalid input accepted")
	}
}
