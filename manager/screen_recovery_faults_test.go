package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func TestRecoveryRejectsUnmatchedOrInvalidPendingEvidence(t *testing.T) {
	for _, outcome := range []screendelivery.Outcome{screendelivery.PendingConfirmed, screendelivery.PendingUnconfirmed, 255} {
		sender := &recoveringSender{ready: func(context.Context) (screendelivery.Readiness, error) {
			return screendelivery.Readiness{Pending: outcome}, nil
		}, send: func(context.Context, display.Frame) error { t.Error("unexpected send"); return nil }}
		p, s, clock := recoveryPump(t, sender)
		submitCycle(t, s, clock, "scene")
		if err := p.Run(context.Background()); !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
		if s.Status().InFlight != 0 || s.Status().Confirmed != 0 {
			t.Fatal(s.Status())
		}
	}
}

func TestRecoveryFailureReleasesUnconfirmedCycle(t *testing.T) {
	for _, pending := range []bool{false, true} {
		failure := errors.New("operator action required")
		sent := false
		sender := &recoveringSender{ready: func(context.Context) (screendelivery.Readiness, error) {
			if !pending || sent {
				return screendelivery.Readiness{}, failure
			}
			return screendelivery.Readiness{}, nil
		}, send: func(context.Context, display.Frame) error { sent = true; return screendelivery.ErrTransportLost }}
		p, s, clock := recoveryPump(t, sender)
		submitCycle(t, s, clock, "scene")
		if err := p.Run(context.Background()); !errors.Is(err, failure) {
			t.Fatal(err)
		}
		if s.Status().InFlight != 0 || s.Status().RefreshTrusted {
			t.Fatal(s.Status())
		}
	}
}

func TestRecoverySendResyncInvalidatesBeforeSessionReset(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sender := &recoveringSender{}
	p, s, clock := recoveryPump(t, sender)
	submitCycle(t, s, clock, "scene")
	sends := 0
	sender.ready = func(context.Context) (screendelivery.Readiness, error) {
		if sender.resets > 0 && s.Status().InFlight != 0 {
			t.Error("session reset retained old lease")
		}
		return screendelivery.Readiness{}, nil
	}
	sender.send = func(context.Context, display.Frame) error {
		sends++
		if sends == 1 {
			return screendelivery.ErrTransportResync
		}
		cancel()
		return nil
	}
	if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if sender.resets != 1 || sends != 2 || s.Status().FullRefresh.Cycle != 2 {
		t.Fatal(sender.resets, sends, s.Status())
	}
}

func TestRecoverySchedulingErrorsFailWithoutRendering(t *testing.T) {
	sender := &recoveringSender{ready: func(context.Context) (screendelivery.Readiness, error) { return screendelivery.Readiness{}, nil }}
	p, s, clock := recoveryPump(t, sender)
	submitCycle(t, s, clock, "scene")
	clock.tick = -1
	if err := p.Run(context.Background()); err == nil {
		t.Fatal("clock regression accepted")
	}
	closed := make(chan struct{})
	close(closed)
	if err := waitScreenChange(context.Background(), nil, closed, 0, false); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	wake := make(chan struct{}, 1)
	wake <- struct{}{}
	if err := waitScreenChange(context.Background(), nil, wake, 0, false); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryCancelledWaitReleasesExactPendingCycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sent := false
	sender := &recoveringSender{ready: func(ctx context.Context) (screendelivery.Readiness, error) {
		if sent {
			cancel()
			return screendelivery.Readiness{}, ctx.Err()
		}
		return screendelivery.Readiness{}, nil
	}, send: func(context.Context, display.Frame) error { sent = true; return screendelivery.ErrTransportLost }}
	p, s, clock := recoveryPump(t, sender)
	submitCycle(t, s, clock, "scene")
	if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if s.Status().InFlightCycle != 0 || s.Status().RefreshTrusted {
		t.Fatal(s.Status())
	}
}

func TestScreenPumpMaintenanceUsesElapsedClockWithoutCatchup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p, s, clock := recoveryPump(t, &recoveringSender{})
	submitCycle(t, s, clock, "cached")
	sends := 0
	p.sender = screenSender(func(context.Context, display.Frame) error {
		sends++
		if sends == 2 {
			cancel()
		}
		return nil
	})
	p.sleep = func(ctx context.Context, wake, changed <-chan struct{}, delay time.Duration, pending bool) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !pending {
			return errors.New("missing maintenance timer")
		}
		clock.tick += max(delay, time.Second)
		clock.wall = clock.wall.Add(24 * time.Hour)
		return nil
	}
	if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if sends != 2 || clock.tick != 601*time.Second || s.Status().Current != 1 {
		t.Fatal(sends, clock.tick, s.Status())
	}
}

func TestRecoveryUnknownHidesOldProofButKeepsReconcilableCycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sender := &recoveringSender{}
	p, s, clock := recoveryPump(t, sender)
	submitCycle(t, s, clock, "scene")
	confirmCycle(t, s, nextCycle(t, s, clock))
	if err := s.forceLatest(); err != nil {
		t.Fatal(err)
	}
	sent := false
	sender.ready = func(context.Context) (screendelivery.Readiness, error) {
		if !sent {
			return screendelivery.Readiness{}, nil
		}
		status := s.Status()
		if status.Confirmed != 0 || status.RefreshTrusted {
			t.Error("unknown send retained current image proof", status)
		}
		if status.InFlightCycle != 2 || status.FullRefresh.Cycle != 1 {
			t.Error("pending or historical cycle lost", status)
		}
		cancel()
		return screendelivery.Readiness{Pending: screendelivery.PendingConfirmed}, nil
	}
	sender.send = func(context.Context, display.Frame) error { sent = true; return screendelivery.ErrTransportLost }
	if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if !s.Status().RefreshTrusted || s.Status().FullRefresh.Cycle != 2 {
		t.Fatal(s.Status())
	}
}
