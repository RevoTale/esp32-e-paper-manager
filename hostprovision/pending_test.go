package hostprovision

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

func candidateFixture() (provision.Request, provision.Response, Enrollment) {
	c := provision.Config{SSID: "secure", Passphrase: "long-secret", Manager: "tcp://manager.example:9757",
		Timezone: "Europe/Kyiv", DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Auth: provision.AuthWPA3SAE}
	r := provision.Response{Operation: provision.OperationRotate, Code: provision.CodeOK,
		State: provision.StateProvisioned, Generation: 2, DeviceID: c.DeviceID,
		Auth: c.Auth, SSID: c.SSID, Manager: c.Manager, Timezone: c.Timezone}
	return provision.Request{Operation: r.Operation, Config: c}, r,
		Enrollment{DeviceID: c.DeviceID, DeviceKey: c.DeviceKey, Timezone: c.Timezone}
}

func TestStageAndConfirmEnrollment(t *testing.T) {
	request, response, enrollment := candidateFixture()
	for _, existing := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "device.json")
		previous := preparePreviousEnrollment(t, path, enrollment, existing)
		pending, err := StageEnrollment(path, enrollment)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = pending.Close() }()
		if existing && !bytes.Equal(mustRead(t, path), previous) {
			t.Fatal("active changed before ACK")
		}
		candidate := mustRead(t, path+".pending")
		assertPrivateEnrollment(t, path+".pending")
		if err = pending.Confirm(request, response); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(mustRead(t, path), candidate) {
			t.Fatal("wrong candidate promoted")
		}
		if _, err = os.Lstat(path + ".pending"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("pending remains: %v", err)
		}
		if err = pending.Confirm(request, response); !errors.Is(err, ErrPromotion) {
			t.Fatalf("second confirmation: %v", err)
		}
	}
}

func assertPrivateEnrollment(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil || !privateFile(info, maxEnrollmentBytes) {
		t.Fatalf("non-private candidate: %v", err)
	}
}

func preparePreviousEnrollment(t *testing.T, path string, enrollment Enrollment, existing bool) []byte {
	t.Helper()
	if !existing {
		return nil
	}
	enrollment.DeviceKey[0]++
	if err := SaveEnrollment(path, enrollment); err != nil {
		t.Fatal(err)
	}
	return mustRead(t, path)
}

func TestStageRejectsExistingPendingWithoutChangingIt(t *testing.T) {
	_, _, enrollment := candidateFixture()
	for _, kind := range []string{"file", "symlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "device.json")
			makeRegistryObject(t, path+".pending", kind)
			before, err := os.Lstat(path + ".pending")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = StageEnrollment(path, enrollment); !errors.Is(err, ErrPending) {
				t.Fatalf("existing=%v", err)
			}
			after, err := os.Lstat(path + ".pending")
			if err != nil || !os.SameFile(before, after) {
				t.Fatalf("pending replaced: %v", err)
			}
		})
	}
}

func TestStageRejectsUnsafeActiveFiles(t *testing.T) {
	_, _, enrollment := candidateFixture()
	for _, kind := range []string{"symlink", "directory", "empty", "large", "public", "invalid"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "device.json")
			makeRegistryObject(t, path, kind)
			if _, err := StageEnrollment(path, enrollment); !errors.Is(err, ErrStaging) {
				t.Fatalf("unsafe active=%v", err)
			}
			if _, err := os.Lstat(path + ".pending"); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("pending created: %v", err)
			}
		})
	}
}

func TestStageRejectsInvalidInputAndMissingDirectory(t *testing.T) {
	_, _, enrollment := candidateFixture()
	for _, path := range []string{"", filepath.Join(t.TempDir(), "missing", "file")} {
		if _, err := StageEnrollment(path, enrollment); !errors.Is(err, ErrStaging) {
			t.Fatal(err)
		}
	}
	for _, invalid := range []Enrollment{{}, {DeviceID: enrollment.DeviceID},
		{DeviceID: enrollment.DeviceID, DeviceKey: enrollment.DeviceKey},
		{DeviceID: enrollment.DeviceID, DeviceKey: enrollment.DeviceKey, Timezone: string(bytes.Repeat([]byte{'a'}, 65))}} {
		if _, err := StageEnrollment(filepath.Join(t.TempDir(), "file"), invalid); !errors.Is(err, ErrStaging) {
			t.Fatal(err)
		}
	}
}

func makeRegistryObject(t *testing.T, path, kind string) {
	t.Helper()
	var err error
	switch kind {
	case "symlink":
		err = os.Symlink("missing-target", path)
	case "directory":
		err = os.Mkdir(path, 0o700)
	case "empty":
		err = os.WriteFile(path, nil, 0o600)
	case "large":
		err = os.WriteFile(path, bytes.Repeat([]byte{'x'}, maxEnrollmentBytes+1), 0o600)
	case "public":
		err = os.WriteFile(path, []byte("private"), 0o644)
	default:
		err = os.WriteFile(path, []byte("private"), 0o600)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
