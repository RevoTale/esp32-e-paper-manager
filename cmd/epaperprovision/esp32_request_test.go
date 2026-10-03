package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
	"github.com/RevoTale/esp32-e-paper-manager/provision"
	"github.com/RevoTale/esp32-e-paper-manager/provisionclient"
)

func TestESP32BuildRequestStagesMatchingWPA2Enrollment(t *testing.T) {
	codec := provision.ESP32Codec()
	stream := &bytes.Buffer{}
	client, err := provisionclient.NewFor(stream, codec)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "device.json")
	request, pending, err := buildRequest(client, provision.OperationProvision, path, strings.NewReader(provisionInput))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pending.Close(); err != nil {
			t.Error(err)
		}
	})
	if request.Config.Auth != provision.AuthWPA2PSK || request.Config.ValidateFor(codec) != nil || request.Config.Validate() == nil {
		t.Fatal("buildRequest did not retain explicit ESP32 policy")
	}
	if stream.Len() != 0 {
		t.Fatal("request preparation wrote to device")
	}
	checkESP32PendingCandidate(t, request, path)
	wire := make([]byte, provision.RequestSize)
	if err := codec.EncodeRequest(wire, request); err != nil || wire[16] != 2 {
		t.Fatal("incorrect WPA2 wire encoding", err)
	}
}

func checkESP32PendingCandidate(t *testing.T, request provision.Request, path string) {
	t.Helper()
	enrollment, err := hostprovision.LoadEnrollment(path + ".pending")
	if err != nil || enrollment.DeviceID != request.Config.DeviceID || enrollment.DeviceKey != request.Config.DeviceKey || enrollment.Timezone != request.Config.Timezone {
		t.Fatal("pending enrollment differs from request", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unconfirmed enrollment promoted", err)
	}
}
