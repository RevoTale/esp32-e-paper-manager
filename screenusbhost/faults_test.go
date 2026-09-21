package screenusbhost

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestInvalidConfigurationAndCancelledReadiness(t *testing.T) {
	for _, size := range []display.Size{{}, {Width: 2049, Height: 1}, {Width: 2048, Height: 2048}} {
		if _, err := New("worker", "port", size); !errors.Is(err, ErrConfiguration) {
			t.Fatal(size, err)
		}
	}
	for _, args := range [][2]string{{"", "port"}, {"worker", ""}} {
		if _, err := New(args[0], args[1], display.Size{Width: 8, Height: 1}); !errors.Is(err, ErrConfiguration) {
			t.Fatal(err)
		}
	}
	s, _, _, _ := usbFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.WaitReady(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.Send(ctx, display.Frame{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestClosedSenderAndMissingExecutableDoNotRetry(t *testing.T) {
	s, _, _, f := usbFixture(t)
	if err := s.Send(context.Background(), f); !errors.Is(err, screendelivery.ErrTransportLost) {
		t.Fatal(err)
	}
	s.start = func() (worker, error) { return nil, ErrWorker }
	if _, err := s.WaitReady(context.Background()); !errors.Is(err, ErrWorker) {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.WaitReady(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if err := s.waitRetry(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestUSBRebootAndGeometryChangeFailClosed(t *testing.T) {
	s, _, sink, _ := usbFixture(t)
	if _, err := s.WaitReady(context.Background()); err != nil {
		t.Fatal(err)
	}
	caps := s.caps
	if err := s.disconnect(); err != nil {
		t.Fatal(err)
	}
	d, err := screenlink.New(caps, [16]byte{2}, sink, time.Second, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.start = func() (worker, error) { return &testWorker{link: d.Open(), done: make(chan struct{})}, nil }
	if _, err = s.WaitReady(context.Background()); !errors.Is(err, screendelivery.ErrTransportResync) {
		t.Fatal(err)
	}
	s.ResetSession()
	if _, err = s.WaitReady(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err = s.disconnect(); err != nil {
		t.Fatal(err)
	}
	s.ResetSession()
	s.size.Width = 8
	if _, err = s.WaitReady(context.Background()); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
}

func TestReadinessPollingDoesNotRenewCooldown(t *testing.T) {
	s, _, _, _ := usbFixture(t)
	r, err := s.WaitReady(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	floor := r.NotBefore
	s.now = func() time.Time { return floor.Add(time.Hour) }
	r, err = s.WaitReady(context.Background())
	if err != nil || r.NotBefore != floor {
		t.Fatal(r, err)
	}
	s.cooldown()
	newFloor := s.floor
	s.now = func() time.Time { return floor }
	s.cooldown()
	if s.floor != newFloor {
		t.Fatal("floor regressed")
	}
}

type closeFailure struct{ *testWorker }

func (*closeFailure) Close() error { return ErrWorker }

func TestUnreapedWorkerCannotBeReusedAfterReset(t *testing.T) {
	s, d, _, _ := usbFixture(t)
	w := &closeFailure{&testWorker{link: d.Open(), done: make(chan struct{})}}
	s.current = w
	if err := s.disconnect(); !errors.Is(err, ErrWorker) {
		t.Fatal(err)
	}
	s.ResetSession()
	if _, err := s.process(); !errors.Is(err, ErrWorker) {
		t.Fatal("reused unreaped worker", err)
	}
	for n := 0; n < 2; n++ {
		if err := s.Close(); !errors.Is(err, ErrWorker) {
			t.Fatal("close hid reap failure", n, err)
		}
	}
	if err := w.testWorker.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidFrameDoesNotCreatePendingAndRetryWaitCanCancel(t *testing.T) {
	s, _, _, _ := usbFixture(t)
	r, err := s.WaitReady(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return r.NotBefore }
	if err = s.Send(context.Background(), display.Frame{}); !errors.Is(err, screenwire.ErrRecord) || s.client.Pending() {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = s.waitRetry(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if retryable(screenwire.ErrRecord) || !retryable(io.EOF) {
		t.Fatal("bad error class")
	}
}
