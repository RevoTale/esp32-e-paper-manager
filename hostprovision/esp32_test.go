package hostprovision

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

func TestESP32EnrollmentRequiresExplicitConfirmationPolicy(t *testing.T) {
	codec := provision.ESP32Codec()
	input := Input{SSID: "example", Passphrase: "passphrase", Manager: "tcp://manager:1234", Timezone: "Europe/Kyiv"}
	config, enrollment, err := CreateFor(codec, input, nil, bytes.NewReader(bytes.Repeat([]byte{1}, 48)))
	if err != nil || config.Auth != provision.AuthWPA2PSK {
		t.Fatalf("config=%+v err=%v", config.Auth, err)
	}
	pending, err := StageEnrollment(filepath.Join(t.TempDir(), "device.json"), enrollment)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pending.Close(); err != nil {
			t.Error(err)
		}
	})
	request := provision.Request{Operation: provision.OperationProvision, Config: config}
	response := provision.Response{Operation: request.Operation, State: provision.StateProvisioned,
		Generation: 1, Auth: config.Auth, DeviceID: config.DeviceID, SSID: config.SSID, Manager: config.Manager, Timezone: config.Timezone}
	if pending.Confirm(request, response) == nil {
		t.Fatal("default policy accepted WPA2")
	}
	if err := pending.ConfirmFor(codec, request, response); err != nil {
		t.Fatal(err)
	}
}
