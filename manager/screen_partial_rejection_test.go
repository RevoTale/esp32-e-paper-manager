package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func TestUnavailablePartialDoesNotStopPumpOrLoseBaseline(t *testing.T) {
	s, c := partialFixture(t)
	submitCycle(t, s, c, "A")
	confirmCycle(t, s, nextCycle(t, s, c))
	proof := *s.Status().FullRefresh
	c.advance(time.Second)
	submitCycle(t, s, c, "B")
	s.options.Mode = refreshpolicy.Partial
	s.cycles.partials = s.cycles.partialOptions.MaxConsecutive
	r := screenRecovery{pump: &ScreenPump{screen: s}}
	if err := r.deliver(context.Background()); err != nil {
		t.Fatalf("expected per-scene rejection, not pump failure: %v", err)
	}
	status := s.Status()
	if status.Confirmed != 1 || status.InFlight != 0 || !status.RefreshTrusted || *status.FullRefresh != proof {
		t.Fatal(status)
	}
	// Maintenance must use accepted pixels, never silently promote rejected B.
	c.advance(10 * time.Minute)
	d := nextCycle(t, s, c)
	if d.Revision != 1 || d.Region != nil {
		t.Fatal("maintenance promoted rejected partial", d)
	}
	confirmCycle(t, s, d)
	submitCycle(t, s, c, "C")
	d = nextCycle(t, s, c)
	confirmCycle(t, s, d)
	if s.Status().Confirmed != 3 {
		t.Fatal("new submission did not recover", s.Status())
	}
}

func TestPartialRejectionDiagnosticsAreBoundedAndOwned(t *testing.T) {
	for _, area := range []bool{false, true} {
		s, c := partialFixture(t)
		cause, reason := refreshstamp.ErrUnconfirmed, RefreshRequiresFull
		if area {
			wideDamageFixture(s)
			cause, reason = screendelivery.ErrRegionBudget, RefreshRegionLimit
		}
		submitCycle(t, s, c, "A")
		confirmCycle(t, s, nextCycle(t, s, c))
		submitCycle(t, s, c, "B")
		s.options.Mode = refreshpolicy.Partial
		if !area {
			s.cycles.partials = s.cycles.partialOptions.MaxConsecutive
		}
		_, err := s.RenderNext(context.Background(), c.tick, true)
		if !errors.Is(err, ErrPartialUnavailable) || !errors.Is(err, cause) {
			t.Fatal(err)
		}
		assertRefreshRejection(t, s, reason)
		submitCycle(t, s, c, "C")
		if s.Status().RefreshFailure != nil {
			t.Fatal("new submission retained rejection")
		}
	}
}

func assertRefreshRejection(t *testing.T, s *Screen, reason RefreshRejectionReason) {
	t.Helper()
	status := s.Status()
	if status.Failure != nil || status.RefreshFailure == nil || *status.RefreshFailure != (ScreenRefreshFailure{Revision: 2, Reason: reason}) {
		t.Fatal(status)
	}
	status.RefreshFailure.Reason = "caller mutation"
	if s.Status().RefreshFailure.Reason != reason {
		t.Fatal("status aliases internal failure")
	}
}

func TestPartialRejectionDoesNotHideQueueFailure(t *testing.T) {
	s, _ := partialFixture(t)
	// Without a queue lease, a rejection must not mask broken ownership.
	if _, err := s.rejectPartial(1, refreshstamp.ErrUnconfirmed); err == nil || errors.Is(err, ErrPartialUnavailable) {
		t.Fatal(err)
	}
	if s.Status().RefreshFailure != nil {
		t.Fatal("invented rejection without lease")
	}
}
