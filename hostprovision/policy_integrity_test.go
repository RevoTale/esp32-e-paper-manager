package hostprovision

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

func TestPendingAuthPolicyMismatchPreservesBothEnrollments(t *testing.T) {
	for _, mismatch := range []string{"default-codec", "request-auth", "response-auth", "storage-error"} {
		t.Run(mismatch, func(t *testing.T) { checkPendingAuthMismatch(t, mismatch) })
	}
}

func checkPendingAuthMismatch(t *testing.T, mismatch string) {
	t.Helper()
	request, response, enrollment := candidateFixture()
	request.Config.Auth, response.Auth = provision.AuthWPA2PSK, provision.AuthWPA2PSK
	codec := provision.ESP32Codec()
	path := filepath.Join(t.TempDir(), "device.json")
	previous := preparePreviousEnrollment(t, path, enrollment, true)
	pending, err := StageEnrollment(path, enrollment)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pending.Close(); err != nil {
			t.Error(err)
		}
	})
	candidate := mustRead(t, path+".pending")
	switch mismatch {
	case "default-codec":
		codec = provision.Codec{}
	case "request-auth":
		request.Config.Auth = provision.AuthWPA3SAE
	case "response-auth":
		response.Auth = provision.AuthWPA3SAE
	case "storage-error":
		response.Code = provision.CodeStorage
	}
	if err := pending.ConfirmFor(codec, request, response); !errors.Is(err, ErrAcknowledgement) {
		t.Fatal("unconfirmed auth policy promoted", err)
	}
	if !bytes.Equal(previous, mustRead(t, path)) || !bytes.Equal(candidate, mustRead(t, path+".pending")) {
		t.Fatal("auth mismatch changed enrollment files")
	}
}
