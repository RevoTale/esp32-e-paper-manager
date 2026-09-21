package screenhub

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

func testConfig() Config {
	return Config{ID: securetransport.DeviceID{1}, Key: securetransport.Key{2},
		Size: display.Size{Width: 17, Height: 9}, Profile: 1, ProfileVersion: 1}
}

func TestConfigurationAndClosedHub(t *testing.T) {
	for _, cfg := range []Config{{}, {ID: testConfig().ID}, {ID: testConfig().ID, Key: testConfig().Key}} {
		if _, err := New(cfg); err == nil {
			t.Fatal("invalid config accepted")
		}
	}
	h, err := New(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err = h.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-h.Changed():
	default:
		t.Fatal("close did not wake an idle pump")
	}
	if _, err = h.WaitReady(context.Background()); !errors.Is(err, net.ErrClosed) {
		t.Fatal(err)
	}
	a, b := net.Pipe()
	t.Cleanup(func() { _ = b.Close() })
	if err = h.Offer(context.Background(), a); !errors.Is(err, net.ErrClosed) {
		t.Fatal(err)
	}
}

func TestWaitCancellationAndUnconnectedSend(t *testing.T) {
	h, _ := New(testConfig())
	t.Cleanup(func() { _ = h.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := h.WaitReady(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := h.Send(context.Background(), display.Frame{}); !errors.Is(err, screendelivery.ErrTransportLost) {
		t.Fatal(err)
	}
}

func TestHandshakeSlotsAreBoundedAndCancellationReleases(t *testing.T) {
	h, _ := New(testConfig())
	t.Cleanup(func() { _ = h.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 4)
	for range 4 {
		a, b := net.Pipe()
		t.Cleanup(func() { _ = b.Close() })
		if err := h.reserve(a); err != nil {
			t.Fatal(err)
		}
		go func() { done <- h.authenticate(ctx, a) }()
	}
	a, b := net.Pipe()
	t.Cleanup(func() { _ = b.Close() })
	if err := h.Offer(ctx, a); !errors.Is(err, ErrCapacity) {
		t.Fatal(err)
	}
	cancel()
	for range 4 {
		select {
		case err := <-done:
			if err == nil {
				t.Fatal("cancel accepted")
			}
		case <-time.After(time.Second):
			t.Fatal("handshake did not stop")
		}
	}
}
