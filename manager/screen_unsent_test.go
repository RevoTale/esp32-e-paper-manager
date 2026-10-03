package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func unsentFixture(t *testing.T) (*screenRecovery, *cycleClock) {
	t.Helper()
	s, c := partialFixture(t)
	submitCycle(t, s, c, "A")
	confirmCycle(t, s, nextCycle(t, s, c))
	p := dispatchPump(t, s, c, &dispatchSender{onSend: func(refreshpolicy.Mode) error {
		t.Fatal("unexpected transmission")
		return nil
	}})
	r := &screenRecovery{pump: p}
	r.completed()
	c.advance(2 * time.Second)
	submitCycle(t, s, c, "B")
	s.cycles.partials = s.cycles.partialOptions.MaxConsecutive
	if err := r.deliver(context.Background()); err != nil {
		t.Fatal(err)
	}
	return r, c
}

func TestReadinessDoesNotReconcileUnsentCycle(t *testing.T) {
	r, c := unsentFixture(t)
	s := r.pump.screen
	id := s.Status().InFlightCycle
	r.transport = &recoveringSender{ready: func(context.Context) (screendelivery.Readiness, error) {
		return screendelivery.Readiness{NotBefore: c.wall.Add(time.Minute)}, nil
	}}
	if err := r.ready(context.Background()); err != nil || s.Status().InFlightCycle != id || !r.unsent {
		t.Fatal(err, s.Status())
	}
	if r.remaining() != time.Minute {
		t.Fatal("readiness floor lost", r.remaining())
	}
	if err := r.reconcile(screendelivery.PendingConfirmed); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if err := r.resolve(false); err != nil || s.Status().Confirmed != 1 || !s.Status().RefreshTrusted {
		t.Fatal(err, s.Status())
	}
	if _, err := s.cycles.tracker.ForPartial(); err != nil {
		t.Fatal(err)
	}
}

func TestUnsentResetRequiresFreshFullBaseline(t *testing.T) {
	r, _ := unsentFixture(t)
	sender := &recoveringSender{}
	r.transport = sender
	if err := r.reset(); err != nil {
		t.Fatal(err)
	}
	s := r.pump.screen
	if sender.resets != 1 || s.Status().RefreshTrusted || s.Status().Confirmed != 0 || r.unsent || !s.cycles.resync {
		t.Fatal(sender.resets, s.Status(), r.unsent)
	}
}

func TestUnsentPartialBecomesFullWhenMaintenanceDue(t *testing.T) {
	s, c := partialFixture(t)
	submitCycle(t, s, c, "A")
	confirmCycle(t, s, nextCycle(t, s, c))
	p := dispatchPump(t, s, c, &dispatchSender{onSend: func(refreshpolicy.Mode) error {
		t.Fatal("unexpected transmission")
		return nil
	}})
	r := screenRecovery{pump: p}
	r.completed()
	submitCycle(t, s, c, "B")
	if err := r.deliver(context.Background()); err != nil || r.pending.Region == nil {
		t.Fatal(err)
	}
	c.advance(DefaultMaintenanceInterval)
	if _, pending, err := r.waitReady(); err != nil || !pending || r.unsent || !s.cycles.resync {
		t.Fatal(err, pending, r.unsent, s.cycles.resync)
	}
	if options := s.pendingOptions(); options.Mode != refreshpolicy.Full || options.Priority != refreshpolicy.Normal {
		t.Fatal(options)
	}
}

func TestPumpPartialConfigurationRejectsMissingContracts(t *testing.T) {
	s, c := partialFixture(t)
	p := dispatchPump(t, s, c, &dispatchSender{})
	policy := p.partialPolicy
	if err := p.ConfigurePartial(refreshpolicy.Policy{}); err == nil || p.partialPolicy != policy {
		t.Fatal(err, p.partialPolicy)
	}
	s.pumping.Store(true)
	if err := p.ConfigurePartial(policy); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	s.pumping.Store(false)
	p.sender = &partialSender{}
	if err := p.ConfigurePartial(policy); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	p.urgent = 0
	if err := p.ConfigurePartial(policy); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
}
