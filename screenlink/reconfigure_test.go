package screenlink

import (
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamsession"
)

func TestReconfigureRevokesEvenUnboundAuthenticatedConnections(t *testing.T) {
	d, active, sink := fixture(t)
	unbound := d.Open()
	bind(t, active, 2, true)
	upload(t, active, 1)
	if err := d.Reconfigure([16]byte{8}); err != nil {
		t.Fatal(err)
	}
	var raw [screenwire.HeaderSize]byte
	_, _ = screenwire.Encode(raw[:], screenwire.Record{Kind: screenwire.Hello})
	for _, old := range []*Connection{active, unbound} {
		if err := old.Push(raw[:], time.Second, func([]byte) error {
			t.Error("revoked connection replied")
			return nil
		}); !errors.Is(err, streamsession.ErrLease) {
			t.Fatal(err)
		}
	}
	if d.caps.DeviceID != [16]byte{8} || sink.aborts != 1 {
		t.Fatal(d.caps, sink)
	}
	s := exchange(t, d.Open(), screenwire.Record{Kind: screenwire.Hello}, time.Second)
	if s.Generation != 2 || s.Code != screenwire.CodeOK {
		t.Fatal(s)
	}
}

func TestReconfigureDoesNotWrapOrPublishIdentityOnFailure(t *testing.T) {
	d, c, _ := fixture(t)
	d.authentication = ^uint64(0)
	if err := d.Reconfigure([16]byte{8}); !errors.Is(err, streamsession.ErrLease) {
		t.Fatal(err)
	}
	if !d.disabled || d.caps.DeviceID != [16]byte{} {
		t.Fatal("identity published after failure")
	}
	if err := c.Push(nil, 0, func([]byte) error { return nil }); !errors.Is(err, streamsession.ErrLease) {
		t.Fatal("disabled device accepted old context", err)
	}
}
