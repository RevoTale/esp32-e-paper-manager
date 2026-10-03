package screenclient

import (
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestReadHealthSendsOnlyOneUnboundQuery(t *testing.T) {
	client, device, _, sink, _ := fixture(t)
	health := screenwire.HealthStatus{Version: 1, State: 5, UptimeSeconds: 12}
	device.SetHealth(func() screenwire.HealthStatus { return health })
	observer := &transport{connection: device.Open(), now: 24 * time.Hour}
	response, err := ReadHealth(observer)
	if err != nil || response.Health != health || response.Status.Generation != client.lease.Generation {
		t.Fatal(response, err)
	}
	if observer.wireBytes != 2*screenwire.HeaderSize+screenwire.StatusSize+screenwire.HealthSize || sink.commits != 0 || client.Pending() {
		t.Fatal("health acquired or sent an image", observer.wireBytes, sink, client.Pending())
	}
}

func TestReadHealthPropagatesMissingStreamAndProvider(t *testing.T) {
	if _, err := ReadHealth(nil); !errors.Is(err, ErrConnection) {
		t.Fatal(err)
	}
	_, device, _, _, _ := fixture(t)
	_, err := ReadHealth(&transport{connection: device.Open()})
	var remote RemoteError
	if !errors.As(err, &remote) || remote.Status.Code != screenwire.CodeConfig {
		t.Fatal(err)
	}
}
