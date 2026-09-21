package provisionclient

import (
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

func TestWrongBoardResponsePoisonsClientWithoutAnotherWrite(t *testing.T) {
	for _, codec := range []provision.Codec{{}, provision.ESP32Codec()} {
		other := provision.Codec{}
		if codec.AuthMode() == provision.AuthWPA3SAE {
			other = provision.ESP32Codec()
		}
		reply := make([]byte, provision.ResponseSize)
		response := provision.Response{Operation: provision.OperationInspect, State: provision.StateProvisioned,
			Generation: 1, Auth: other.AuthMode(), DeviceID: [16]byte{1}, SSID: "test",
			Manager: "tcp://manager:1234", Timezone: "Europe/Kyiv"}
		if err := other.EncodeResponse(reply, response); err != nil {
			t.Fatal(err)
		}
		stream := &loopback{reply: reply}
		client, err := NewFor(stream, codec)
		if err != nil {
			t.Fatal(err)
		}
		request := provision.Request{Operation: provision.OperationInspect}
		if _, err = client.Execute(request); !errors.Is(err, provision.ErrInvalidConfig) {
			t.Fatal("wrong board accepted", err)
		}
		if _, err = client.Execute(request); !errors.Is(err, ErrConfiguration) {
			t.Fatal("ambiguous connection reused", err)
		}
		if stream.request.Len() != provision.RequestSize {
			t.Fatal("poisoned connection wrote again")
		}
	}
}
