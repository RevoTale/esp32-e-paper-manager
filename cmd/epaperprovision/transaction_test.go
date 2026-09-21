package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
	"github.com/RevoTale/esp32-e-paper-manager/provision"
	"go.bug.st/serial"
)

const provisionInput = `{"ssid":"secure","passphrase":"long-secret","manager":"tcp://manager.example:9757"}`

func TestUncertainRotationPreservesActiveAndPending(t *testing.T) {
	cases := []struct {
		name   string
		device *fakeSerial
	}{
		{"lost ACK", &fakeSerial{state: provision.StateProvisioned, loseACK: true}},
		{"partial write", &fakeSerial{state: provision.StateProvisioned, writeFail: true}},
		{"rejected", &fakeSerial{state: provision.StateProvisioned, mutate: func(r *provision.Response) {
			if r.Operation == provision.OperationRotate {
				r.Code = provision.CodeStorage
			}
		}}},
		{"different metadata", &fakeSerial{state: provision.StateProvisioned, mutate: func(r *provision.Response) {
			if r.Operation == provision.OperationRotate {
				r.SSID = "unexpected"
			}
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, previous := activeEnrollment(t)
			original := openSerial
			t.Cleanup(func() { openSerial = original })
			openSerial = func(string, *serial.Mode) (serialDevice, error) { return tc.device, nil }
			err := run([]string{"-port", "test", "-registry", path, "rotate"}, strings.NewReader(provisionInput), io.Discard)
			if err == nil {
				t.Fatal("uncertain rotation accepted")
			}
			if !strings.Contains(err.Error(), "recovery=") {
				t.Fatalf("missing recovery: %v", err)
			}
			assertFileBytes(t, path, previous)
			if _, err = os.Stat(path + ".pending"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestExistingPendingBlocksNewProvision(t *testing.T) {
	path, previous := activeEnrollment(t)
	pending := []byte("private unresolved candidate")
	if err := os.WriteFile(path+".pending", pending, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := buildRequest(nil, provision.OperationProvision, path, strings.NewReader(provisionInput)); err == nil {
		t.Fatal("existing pending enrollment overwritten")
	}
	assertFileBytes(t, path, previous)
	assertFileBytes(t, path+".pending", pending)
}

func TestConfirmedRotationPromotesMatchingCandidate(t *testing.T) {
	path, previous := activeEnrollment(t)
	original := openSerial
	t.Cleanup(func() { openSerial = original })
	device := &fakeSerial{state: provision.StateProvisioned}
	device.observe = func(request provision.Request) {
		if request.Operation != provision.OperationRotate {
			return
		}
		assertFileBytes(t, path, previous)
		candidate, err := os.ReadFile(path + ".pending")
		if err != nil || bytes.Equal(candidate, previous) {
			t.Fatalf("candidate missing before USB mutation: %v", err)
		}
	}
	openSerial = func(string, *serial.Mode) (serialDevice, error) { return device, nil }
	if err := run([]string{"-port", "test", "-registry", path, "rotate"}, strings.NewReader(provisionInput), io.Discard); err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(path)
	if err != nil || bytes.Equal(current, previous) {
		t.Fatalf("active unchanged: %v", err)
	}
	if _, err = os.Stat(path + ".pending"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pending remains: %v", err)
	}
}

func TestRunRejectsFlagsOperationAndUnavailablePort(t *testing.T) {
	for _, args := range [][]string{{"-missing"}, {"unknown"}} {
		if err := run(args, nil, io.Discard); err == nil {
			t.Fatal("invalid command accepted")
		}
	}
	original := openSerial
	t.Cleanup(func() { openSerial = original })
	openSerial = func(string, *serial.Mode) (serialDevice, error) { return nil, io.ErrClosedPipe }
	if err := run([]string{"-port", "test", "inspect"}, nil, io.Discard); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}

func activeEnrollment(t *testing.T) (string, []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "device.json")
	enrollment := hostprovision.Enrollment{DeviceID: [16]byte{1}, DeviceKey: [32]byte{9}, Timezone: "Europe/Kyiv"}
	if err := hostprovision.SaveEnrollment(path, enrollment); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, data
}

func assertFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, want) {
		t.Fatalf("file changed: %v", err)
	}
}
