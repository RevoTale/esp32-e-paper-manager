package interop

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/provision"
	"github.com/RevoTale/esp32-e-paper-manager/provisionclient"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
)

type peerStream struct {
	io.Reader
	io.Writer
}

func TestGoClientDeliversTwoFullFramesToNativePanelCore(t *testing.T) {
	for _, advanced := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "refresh-policy"}[advanced], func(t *testing.T) { testFrames(t, advanced) })
	}
}

func testFrames(t *testing.T, advanced bool) {
	t.Helper()
	peer := startNativePeerNamed(t, "EP_RECEIVER_CLI")
	client, err := screenclient.New(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caps, err := client.Connect(peerStream{peer.output, peer.input})
	if err != nil {
		t.Fatal(err)
	}
	if caps.Width != 800 || caps.Height != 480 || caps.Passes != 2 {
		t.Fatal(caps)
	}
	for _, pixel := range []byte{0xa5, 0x5a} {
		frame := testFrame(t, pixel)
		if err = client.Rebind(); err != nil {
			t.Fatal(err)
		}
		if advanced {
			err = client.SendWithOptions(frame, refreshpolicy.Options{Priority: refreshpolicy.Urgent, Mode: refreshpolicy.Full}, refreshpolicy.Policy{Normal: 180 * time.Second, Urgent: 30 * time.Second})
		} else {
			err = client.Send(frame)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	peer.finish()
}

func testFrame(t *testing.T, pixel byte) display.Frame {
	t.Helper()
	frame, err := display.NewFrame(display.Size{Width: 800, Height: 480}, 100, bytes.Repeat([]byte{pixel}, 48000))
	if err != nil {
		t.Fatal(err)
	}
	return frame
}

func TestGoWPA2ProvisioningRoundTripNativeFlash(t *testing.T) {
	for _, correlated := range []bool{false, true} {
		t.Run(map[bool]string{false: "v2", true: "v3"}[correlated], func(t *testing.T) {
			testProvisioningRoundTrip(t, correlated)
		})
	}
}

func testProvisioningRoundTrip(t *testing.T, correlated bool) {
	t.Helper()
	peer := startNativePeerNamed(t, "EP_RECEIVER_CLI")
	client, err := provisionclient.NewFor(peerStream{peer.output, peer.input}, provision.ESP32Codec())
	if correlated {
		client, err = provisionclient.NewCorrelated(peerStream{peer.output, peer.input})
	}
	if err != nil {
		t.Fatal(err)
	}
	config := provision.Config{Auth: provision.AuthWPA2PSK, SSID: "test", Passphrase: "passphrase",
		Manager: "tcp://manager:000000000000001", Timezone: "Europe/Kyiv", DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}}
	for _, operation := range []provision.Operation{provision.OperationProvision, provision.OperationInspect, provision.OperationRotate, provision.OperationErase} {
		request := provision.Request{Operation: operation, Config: config}
		r, err := client.Execute(request)
		if err != nil {
			t.Fatal(err)
		}
		if operation == provision.OperationErase {
			if r.State != provision.StateBlank {
				t.Fatal(r.State)
			}
		} else if r.Auth != provision.AuthWPA2PSK || r.DeviceID != config.DeviceID {
			t.Fatal("metadata mismatch")
		}
	}
	peer.finish()
}
