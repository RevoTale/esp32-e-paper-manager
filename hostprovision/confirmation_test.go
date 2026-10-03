package hostprovision

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

func TestConfirmRejectsEveryUnconfirmedField(t *testing.T) {
	changes := []func(*provision.Request, *provision.Response){
		func(q *provision.Request, _ *provision.Response) { q.Operation = provision.OperationInspect },
		func(q *provision.Request, _ *provision.Response) { q.Config.Passphrase = "short" },
		func(q *provision.Request, _ *provision.Response) { q.Config.DeviceID[0]++ },
		func(q *provision.Request, _ *provision.Response) { q.Config.DeviceKey[0]++ },
		func(q *provision.Request, _ *provision.Response) { q.Config.Timezone = "UTC" },
		func(_ *provision.Request, r *provision.Response) { r.Operation = provision.OperationProvision },
		func(_ *provision.Request, r *provision.Response) { r.Code = provision.CodeStorage },
		func(_ *provision.Request, r *provision.Response) { r.State = provision.StateUnknown },
		func(_ *provision.Request, r *provision.Response) { r.Generation = 0 },
		func(_ *provision.Request, r *provision.Response) { r.DeviceID[0]++ },
		func(_ *provision.Request, r *provision.Response) { r.Auth = 0 },
		func(_ *provision.Request, r *provision.Response) { r.SSID = "other" },
		func(_ *provision.Request, r *provision.Response) { r.Manager = "tcp://other:1234" },
		func(_ *provision.Request, r *provision.Response) { r.Timezone = "UTC" },
	}
	for index, change := range changes {
		request, response, enrollment := candidateFixture()
		path := filepath.Join(t.TempDir(), "device.json")
		pending, err := StageEnrollment(path, enrollment)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = pending.Close() }()
		change(&request, &response)
		if err = pending.Confirm(request, response); !errors.Is(err, ErrAcknowledgement) {
			t.Fatalf("case=%d: %v", index, err)
		}
		if _, err = os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("active created: %v", err)
		}
		if _, err = os.Stat(path + ".pending"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConfirmRejectsChangedFiles(t *testing.T) {
	for _, change := range []string{"active-created", "active-replaced", "active-removed", "pending-changed", "pending-replaced", "pending-removed", "symlink"} {
		t.Run(change, func(t *testing.T) {
			request, response, enrollment := candidateFixture()
			path := filepath.Join(t.TempDir(), "device.json")
			if change != "active-created" {
				if err := SaveEnrollment(path, enrollment); err != nil {
					t.Fatal(err)
				}
			}
			pending, err := StageEnrollment(path, enrollment)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = pending.Close() }()
			changeRegistry(t, path, change, enrollment)
			if err = pending.Confirm(request, response); !errors.Is(err, ErrPromotion) {
				t.Fatalf("changed accepted: %v", err)
			}
		})
	}
}

func changeRegistry(t *testing.T, path, change string, enrollment Enrollment) {
	t.Helper()
	var err error
	switch change {
	case "active-created", "active-replaced":
		err = SaveEnrollment(path, enrollment)
	case "active-removed":
		err = os.Remove(path)
	case "pending-changed":
		err = os.WriteFile(path+".pending", []byte("changed"), 0o600)
	case "pending-replaced":
		err = SaveEnrollment(path+".pending", enrollment)
	case "pending-removed":
		err = os.Remove(path + ".pending")
	case "symlink":
		if err = os.Remove(path); err == nil {
			err = os.Symlink("device.json.pending", path)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
}
