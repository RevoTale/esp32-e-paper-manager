package manager

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

type dispatchSender struct {
	cadenceSender
	onSend func(refreshpolicy.Mode) error
}

func (s *dispatchSender) ConfigurePartial(policy refreshpolicy.Policy) error {
	cadence := screendelivery.Cadence{}
	return cadence.ConfigurePartial(policy)
}

func (s *dispatchSender) SendWithOptions(_ context.Context, _ display.Frame, o refreshpolicy.Options) error {
	return s.onSend(o.Mode)
}

func (s *dispatchSender) SendRegion(_ context.Context, _ screendelivery.RegionPlan, o refreshpolicy.Options) error {
	return s.onSend(o.Mode)
}

func dispatchPump(t *testing.T, s *Screen, c *cycleClock, sender *dispatchSender) *ScreenPump {
	t.Helper()
	p, err := NewScreenPumpWithPolicy(s, sender, refreshpolicy.Policy{Normal: 30 * time.Second, Urgent: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.ConfigurePartial(refreshpolicy.Policy{Normal: 2 * time.Second, Urgent: time.Second}); err != nil {
		t.Fatal(err)
	}
	p.now = func() time.Time { return c.wall }
	p.sleep = func(ctx context.Context, _, _ <-chan struct{}, delay time.Duration, pending bool) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !pending || delay <= 0 {
			t.Fatalf("unexpected unbounded/spinning wait: pending=%v delay=%v", pending, delay)
		}
		c.advance(delay)
		return nil
	}
	return p
}

func TestPumpActualModeCadenceAndDispatchTimestamp(t *testing.T) {
	s, c := partialFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var times []time.Duration
	var modes []refreshpolicy.Mode
	sender := &dispatchSender{onSend: func(mode refreshpolicy.Mode) error {
		times, modes = append(times, c.tick), append(modes, mode)
		if len(times) == 4 {
			cancel()
		} else {
			submitCycle(t, s, c, string(rune('A'+len(times))))
		}
		return nil
	}}
	p := dispatchPump(t, s, c, sender)
	submitCycle(t, s, c, "A")
	if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	want := []time.Duration{30 * time.Second, 32 * time.Second, 34 * time.Second, 64 * time.Second}
	if !reflect.DeepEqual(times, want) || !reflect.DeepEqual(modes, []refreshpolicy.Mode{refreshpolicy.Full, refreshpolicy.Partial, refreshpolicy.Partial, refreshpolicy.Full}) {
		t.Fatal(times, modes)
	}
	if proof := s.Status().FullRefresh; proof == nil || !proof.Started.Equal(c.wall) || s.Status().Confirmed != 4 {
		t.Fatal("timestamp predates actual dispatch", s.Status())
	}
}

func TestPumpSupersedesUnsentFullWithoutLosingBaseline(t *testing.T) {
	s, c := partialFixture(t)
	submitCycle(t, s, c, "A")
	confirmCycle(t, s, nextCycle(t, s, c))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var modes []refreshpolicy.Mode
	p := dispatchPump(t, s, c, &dispatchSender{onSend: func(mode refreshpolicy.Mode) error {
		modes = append(modes, mode)
		cancel()
		return nil
	}})
	r := screenRecovery{pump: p}
	r.completed()
	c.advance(2 * time.Second)
	submitCycle(t, s, c, "B")
	s.cycles.partials = s.cycles.partialOptions.MaxConsecutive
	if err := r.deliver(ctx); err != nil || len(modes) != 0 {
		t.Fatal("fallback full sent at partial deadline", err, modes)
	}
	s.cycles.partials = 0
	submitCycle(t, s, c, "C")
	if err := r.wait(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.deliver(ctx); err != nil || !reflect.DeepEqual(modes, []refreshpolicy.Mode{refreshpolicy.Partial}) || s.Status().Confirmed != 3 {
		t.Fatal(err, modes, s.Status())
	}
}
