package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
)

func TestUnsentCannotBeConfirmedOrDiscardedByAnotherCycle(t *testing.T) {
	r, _ := unsentFixture(t)
	s := r.pump.screen
	d := r.pending
	wrong := d
	wrong.Cycle++
	if err := s.discardUnsent(wrong); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if err := r.resolve(true); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if r.pending.Cycle != d.Cycle || !r.unsent {
		t.Fatal("invalid confirmation destroyed unsent lease")
	}
	if err := r.resolve(false); err != nil || s.Status().InFlight != 0 {
		t.Fatal(err, s.Status())
	}
}

func TestDispatchCancellationPreservesUnsentLeaseForCleanup(t *testing.T) {
	r, _ := unsentFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.deliver(ctx); !errors.Is(err, context.Canceled) || !r.unsent {
		t.Fatal(err, r.unsent)
	}
	if err := r.resolve(false); err != nil || !r.pump.screen.Status().RefreshTrusted {
		t.Fatal(err)
	}
}

func TestDispatchRejectsInvalidClockBeforeDeviceIO(t *testing.T) {
	r, c := unsentFixture(t)
	c.advance(time.Minute)
	c.wall = time.Time{}
	if err := r.deliver(context.Background()); !errors.Is(err, refreshstamp.ErrTime) {
		t.Fatal(err)
	}
	if status := r.pump.screen.Status(); status.InFlight != 0 || status.Confirmed != 1 {
		t.Fatal(status)
	}
}

func TestUnsentTrackerMismatchStopsBeforeDeviceIO(t *testing.T) {
	for _, discard := range []bool{false, true} {
		r, c := unsentFixture(t)
		r.pump.screen.cycles.tracker.Invalidate()
		c.advance(time.Minute)
		var err error
		if discard {
			err = r.resolve(false)
		} else {
			err = r.deliver(context.Background())
		}
		if !errors.Is(err, refreshstamp.ErrCycle) {
			t.Fatal(err)
		}
	}
}
