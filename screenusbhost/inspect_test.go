package screenusbhost

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestInspectReadsNumericHealthWithoutTakingLease(t *testing.T) {
	s, d, sink, _ := usbFixture(t)
	want := screenwire.HealthStatus{Version: 1, State: 7, LastFailure: 7, Failures: 1, UptimeSeconds: 42}
	d.SetHealth(func() screenwire.HealthStatus { return want })
	r, err := s.Inspect(context.Background())
	if err != nil || r.Health != want || r.Capabilities.Profile != 99 {
		t.Fatal(r, err)
	}
	if r.Status.Generation != 0 || s.bound || s.client.Pending() || sink.commits != 0 {
		t.Fatal("inspect took ownership")
	}
}

func TestInspectRejectsMissingHealthAndUnavailableWorker(t *testing.T) {
	s, _, _, _ := usbFixture(t)
	if _, err := s.Inspect(context.Background()); err == nil {
		t.Fatal("missing provider accepted")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Inspect(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}

func TestCancelledInspectDoesNotOpenUSB(t *testing.T) {
	s, _, _, _ := usbFixture(t)
	opened := false
	s.start = func() (worker, error) { opened = true; return nil, ErrWorker }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Inspect(ctx); !errors.Is(err, context.Canceled) || opened {
		t.Fatal(err, opened)
	}
}

func TestIdleWakeDetectsRebootWithoutAuthorWorkOrUpload(t *testing.T) {
	s, _, sink, _ := usbFixture(t)
	if _, err := s.WaitReady(context.Background()); err != nil {
		t.Fatal(err)
	}
	old := s.current.(*testWorker)
	if err := old.link.Disconnect(); err != nil {
		t.Fatal(err)
	}
	d, err := screenlink.New(s.caps, [16]byte{8}, sink, time.Second, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	old.link = d.Open()
	ticks := make(chan time.Time)
	watchDone := make(chan struct{})
	go func() { s.watchTicks(ticks); close(watchDone) }()
	ticks <- time.Now()
	awaitSignal(t, s.Changed())
	if _, err = s.WaitReady(context.Background()); !errors.Is(err, screendelivery.ErrTransportResync) {
		t.Fatal(err)
	}
	if sink.commits != 0 {
		t.Fatal("probe refreshed")
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	awaitSignal(t, watchDone)
}

func awaitSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("missing bounded lifecycle signal")
	}
}
