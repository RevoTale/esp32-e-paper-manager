package screenusbhost

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
)

func TestUSBPolicyRejectsLegacyPeer(t *testing.T) {
	s, _, sink, _ := usbFixture(t)
	if err := s.ConfigureRefresh(refreshpolicy.Policy{Normal: time.Minute, Urgent: time.Second}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.WaitReady(context.Background()); !errors.Is(err, screenclient.ErrUnsupportedRefresh) {
		t.Fatal(err)
	}
	if sink.commits != 0 || s.bound {
		t.Fatal("unsupported peer enabled")
	}
}

func TestUSBDoesNotSilentlyIgnoreOptions(t *testing.T) {
	s, _, _, frame := usbFixture(t)
	r, err := s.WaitReady(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return r.NotBefore }
	if err := s.SendWithOptions(context.Background(), frame, refreshpolicy.Options{Priority: refreshpolicy.Urgent}); !errors.Is(err, screenclient.ErrUnsupportedRefresh) {
		t.Fatal(err)
	}
}
