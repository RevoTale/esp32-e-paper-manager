package screenusbhost

import (
	"context"
	"errors"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"testing"
)

func TestInspectPanelCachedWithoutOwnership(t *testing.T) {
	s, d, sink, _ := usbFixture(t)
	want := screenwire.PanelStatus{Version: 1, State: 2, Cycle: 8}
	d.SetPanelTrace(func() screenwire.PanelStatus { return want })
	r, err := s.InspectPanel(context.Background())
	if err != nil || r != want || s.bound || s.client.Pending() || sink.commits != 0 {
		t.Fatal(r, err)
	}
}

func TestInspectPanelErrors(t *testing.T) {
	s, _, _, _ := usbFixture(t)
	if _, err := s.InspectPanel(context.Background()); err == nil {
		t.Fatal("missing provider accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.InspectPanel(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.InspectPanel(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
