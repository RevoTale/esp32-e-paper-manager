package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
)

func TestScreenCycleFailureNeverAdvancesStampOrRetries(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	submitCycle(t, s, c, "one")
	confirmCycle(t, s, nextCycle(t, s, c))
	history := *s.Status().FullRefresh
	c.advance(DefaultMaintenanceInterval)
	d := nextCycle(t, s, c)
	if err := s.ResolveCycle(d.Cycle, false); err != nil {
		t.Fatal(err)
	}
	status := s.Status()
	if *status.FullRefresh != history || status.RefreshTrusted || status.Confirmed != 0 || status.Delivered != 1 {
		t.Fatal(status)
	}
	if err := s.ResolveCycle(d.Cycle, true); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	c.advance(24 * time.Hour)
	if next := nextCycle(t, s, c); next.Revision != 0 {
		t.Fatal("implicit retry after ambiguity", next)
	}
	submitCycle(t, s, c, "one")
	next := nextCycle(t, s, c)
	if next.Cycle != 3 || next.Revision != 2 {
		t.Fatal(next)
	}
	confirmCycle(t, s, next)
}

func TestScreenCycleMaintenanceRequiresExactPhysicalACK(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	submitCycle(t, s, c, "one")
	confirmCycle(t, s, nextCycle(t, s, c))
	c.advance(DefaultMaintenanceInterval)
	d := nextCycle(t, s, c)
	if err := s.Resolve(d.Revision, true); !errors.Is(err, ErrConflict) {
		t.Fatal("revision-only ACK accepted for physical cycle", err)
	}
	if err := s.ResolveCycle(d.Cycle-1, true); !errors.Is(err, ErrConflict) {
		t.Fatal("stale cycle ACK accepted", err)
	}
	confirmCycle(t, s, d)
}

func TestScreenCycleCompletionClockFailureInvalidatesProof(t *testing.T) {
	for _, monotonic := range []bool{false, true} {
		t.Run(map[bool]string{false: "wall", true: "elapsed"}[monotonic], func(t *testing.T) {
			s, c := cycleFixture(t, renderbatch.Policy{})
			submitCycle(t, s, c, "one")
			confirmCycle(t, s, nextCycle(t, s, c))
			history := *s.Status().FullRefresh
			c.advance(DefaultMaintenanceInterval)
			d := nextCycle(t, s, c)
			expected := refreshstamp.ErrTime
			if monotonic {
				c.tick--
				expected = renderbatch.ErrClock
			} else {
				c.wall = c.wall.Add(-time.Second)
			}
			if err := s.ResolveCycle(d.Cycle, true); !errors.Is(err, expected) {
				t.Fatal(err)
			}
			status := s.Status()
			if *status.FullRefresh != history || status.RefreshTrusted || status.Confirmed != 0 || status.InFlight != 0 {
				t.Fatal("untrusted completion advanced proof", status)
			}
		})
	}
}

func TestScreenCycleRejectsInvalidStartWithoutSending(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	submitCycle(t, s, c, "one")
	confirmCycle(t, s, nextCycle(t, s, c))
	c.tick += DefaultMaintenanceInterval
	c.wall = c.wall.Add(-time.Second)
	if d, err := s.RenderNext(context.Background(), c.tick, true); !errors.Is(err, refreshstamp.ErrTime) || d.Cycle != 0 {
		t.Fatal(d, err)
	}
	if s.Status().InFlight != 0 || s.Status().FullRefresh.Cycle != 1 {
		t.Fatal(s.Status())
	}
}

func TestScreenCycleWallChangesDoNotScheduleMaintenance(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	submitCycle(t, s, c, "one")
	confirmCycle(t, s, nextCycle(t, s, c))
	c.wall = c.wall.Add(48 * time.Hour)
	delay, pending, err := s.wakeDelay(c.tick)
	if err != nil || !pending || delay != DefaultMaintenanceInterval {
		t.Fatal(delay, pending, err)
	}
	if d := nextCycle(t, s, c); d.Revision != 0 {
		t.Fatal(d)
	}
	c.tick = DefaultMaintenanceInterval
	d := nextCycle(t, s, c)
	started := c.wall
	c.advance(time.Minute)
	confirmCycle(t, s, d)
	proof := s.Status().FullRefresh
	if !proof.Started.Equal(started) || !proof.Completed.Equal(c.wall) {
		t.Fatal(proof)
	}
	proof.Cycle = 999
	if s.Status().FullRefresh.Cycle != 2 {
		t.Fatal("mutable historical proof alias")
	}
}

func TestScreenCycleClockMustFollowConcurrentSubmission(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	submitCycle(t, s, c, "one")
	d := nextCycle(t, s, c)
	c.advance(time.Second)
	submitCycle(t, s, c, "two")
	c.tick--
	if err := s.ResolveCycle(d.Cycle, true); !errors.Is(err, renderbatch.ErrClock) {
		t.Fatal("completion accepted regressing observed elapsed clock", err)
	}
}
