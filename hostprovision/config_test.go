package hostprovision

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeCreateAndPrivateEnrollment(t *testing.T) {
	input, err := DecodeInput(bytes.NewBufferString(
		`{"ssid":"secure","passphrase":"long-secret","manager":"tcp://manager.example:9757"}`,
	))
	if err != nil || input.Timezone != "Europe/Kyiv" {
		t.Fatalf("input=%+v err=%v", input, err)
	}
	random := bytes.NewReader(bytes.Repeat([]byte{1}, 48))
	config, enrollment, err := Create(input, nil, random)
	if err != nil || config.DeviceID != enrollment.DeviceID || config.DeviceKey != enrollment.DeviceKey {
		t.Fatalf("config=%+v enrollment=%+v err=%v", config, enrollment, err)
	}
	path := filepath.Join(t.TempDir(), "device.json")
	if err = SaveEnrollment(path, enrollment); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%v err=%v", info.Mode().Perm(), err)
	}
}

func TestCreateRejectsRandomAndInvalidConfiguration(t *testing.T) {
	input := Input{SSID: "secure", Passphrase: "long-secret", Manager: "tcp://manager.example:9757", Timezone: "Europe/Kyiv"}
	if _, _, err := Create(input, nil, io.LimitReader(bytes.NewReader([]byte{1}), 1)); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("short random=%v", err)
	}
	identity := [16]byte{1}
	if _, _, err := Create(input, &identity, bytes.NewReader([]byte{1})); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("short key random=%v", err)
	}
	input.SSID = ""
	if _, _, err := Create(input, nil, bytes.NewReader(bytes.Repeat([]byte{1}, 48))); !errors.Is(err, ErrInput) {
		t.Fatalf("invalid input=%v", err)
	}
	if _, err := DecodeInput(nil); !errors.Is(err, ErrInput) {
		t.Fatalf("nil input=%v", err)
	}
}

func TestRejectsUnknownAndTrailingInput(t *testing.T) {
	for _, input := range []string{
		`{"ssid":"x","unknown":1}`,
		`{"ssid":"x"}{"ssid":"y"}`,
	} {
		if _, err := DecodeInput(bytes.NewBufferString(input)); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
}

func TestSaveEnrollmentValidationAndUnwritablePath(t *testing.T) {
	if err := SaveEnrollment("", Enrollment{}); !errors.Is(err, ErrRegistry) {
		t.Fatalf("empty=%v", err)
	}
	enrollment := Enrollment{DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Timezone: "Europe/Kyiv"}
	path := filepath.Join(t.TempDir(), "missing", "device.json")
	if err := SaveEnrollment(path, enrollment); !errors.Is(err, ErrRegistry) {
		t.Fatalf("unwritable=%v", err)
	}
}

func TestCreatePreservesIdentityAndSaveOverwritesAtomically(t *testing.T) {
	identity := [16]byte{9}
	input := Input{SSID: "secure", Passphrase: "long-secret", Manager: "tcp://manager.example:9757", Timezone: "Europe/Kyiv"}
	_, enrollment, err := Create(input, &identity, bytes.NewReader(bytes.Repeat([]byte{3}, 32)))
	if err != nil || enrollment.DeviceID != identity {
		t.Fatalf("enrollment=%+v err=%v", enrollment, err)
	}
	path := filepath.Join(t.TempDir(), "device.json")
	if err = SaveEnrollment(path, enrollment); err != nil {
		t.Fatal(err)
	}
	enrollment.DeviceKey[0] = 4
	if err = SaveEnrollment(path, enrollment); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err = verifyPrivate(path); !errors.Is(err, ErrRegistry) {
		t.Fatalf("public file=%v", err)
	}
}
