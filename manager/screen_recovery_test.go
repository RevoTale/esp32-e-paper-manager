package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

type recoveringSender struct {
	ready   func(context.Context) (screendelivery.Readiness, error)
	send    func(context.Context, display.Frame) error
	changed chan struct{}
	resets  int
}

func (s *recoveringSender) Send(ctx context.Context, f display.Frame) error { return s.send(ctx, f) }
func (s *recoveringSender) Changed() <-chan struct{}                        { return s.changed }
func (s *recoveringSender) WaitReady(ctx context.Context) (screendelivery.Readiness, error) {
	return s.ready(ctx)
}
func (s *recoveringSender) ResetSession() { s.resets++ }

func TestRecoveryWaitsBeforeRenderingLatestScene(t *testing.T) {
	a, s := newScreenAPI(t)
	_ = apiRequest(a, "PUT", "/v2/screen", a.tag(0), "old")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ready, rendered := false, ""
	s.renderer = screenRenderer(func(context.Context, display.Size, []byte) (display.Frame, error) {
		if !ready {
			t.Error("render preceded transport readiness")
		}
		rendered = s.markup
		return (apiRenderer{}).Render(ctx, display.Size{}, nil)
	})
	sender := &recoveringSender{ready: func(context.Context) (screendelivery.Readiness, error) {
		if !ready {
			_ = apiRequest(a, "PUT", "/v2/screen", a.tag(1), "latest")
		}
		ready = true
		return screendelivery.Readiness{}, nil
	}, send: func(context.Context, display.Frame) error { cancel(); return nil }}
	p, _ := NewScreenPump(s, sender, time.Nanosecond)
	if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if !ready || rendered != "latest" || s.Status().Confirmed != 2 {
		t.Fatal(ready, rendered, s.Status())
	}
}

func TestRecoveryLostACKConfirmsOriginalCycleWithoutResend(t *testing.T) {
	s, clock := cycleFixture(t, renderbatch.Policy{})
	s.elapsedClock = nil
	submitCycle(t, s, clock, "scene")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	reads, sends := 0, 0
	sender := &recoveringSender{ready: func(context.Context) (screendelivery.Readiness, error) {
		reads++
		if reads == 1 {
			return screendelivery.Readiness{}, nil
		}
		if s.Status().InFlightCycle != 1 {
			t.Error("lost exact pending cycle", s.Status())
		}
		cancel()
		return screendelivery.Readiness{Pending: screendelivery.PendingConfirmed}, nil
	}, send: func(context.Context, display.Frame) error { sends++; return screendelivery.ErrTransportLost }}
	p, _ := NewScreenPump(s, sender, time.Nanosecond)
	if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	status := s.Status()
	if sends != 1 || status.InFlight != 0 || status.Confirmed != 1 || status.FullRefresh == nil || status.FullRefresh.Cycle != 1 {
		t.Fatal(sends, status)
	}
}
