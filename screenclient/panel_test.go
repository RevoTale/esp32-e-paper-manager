package screenclient

import (
	"errors"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"testing"
)

func TestReadPanelTraceOneRequestWithoutLease(t *testing.T) {
	c, d, _, sink, _ := fixture(t)
	want := screenwire.PanelStatus{Version: 1, State: 3, Cycle: 4}
	d.SetPanelTrace(func() screenwire.PanelStatus { return want })
	p := &transport{connection: d.Open()}
	r, err := ReadPanelTrace(p)
	if err != nil || r.Panel != want || c.Pending() || sink.commits != 0 {
		t.Fatal(r, err)
	}
	if p.wireBytes != 2*screenwire.HeaderSize+screenwire.StatusSize+screenwire.PanelStatusSize {
		t.Fatal(p.wireBytes)
	}
}

func TestReadPanelTracePropagatesErrors(t *testing.T) {
	if _, err := ReadPanelTrace(nil); !errors.Is(err, ErrConnection) {
		t.Fatal(err)
	}
	_, d, _, _, _ := fixture(t)
	_, err := ReadPanelTrace(&transport{connection: d.Open()})
	var remote RemoteError
	if !errors.As(err, &remote) || remote.Status.Code != screenwire.CodeConfig {
		t.Fatal(err)
	}
}
