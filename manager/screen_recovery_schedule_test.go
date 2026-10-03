package manager

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
)

func TestScreenPumpContinuousAuthoringCannotStarveMaintenance(t *testing.T) {
	s, clock := cycleFixture(t, renderbatch.Policy{Debounce: time.Hour})
	s.cycles.renderer = reservedRenderer{render: markRenderer}
	submitCycle(t, s, clock, "A")
	clock.advance(time.Hour)
	confirmCycle(t, s, nextCycle(t, s, clock))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	latest := byte('A')
	var sent []time.Duration
	sender := screenSender(func(_ context.Context, f display.Frame) error {
		if f.Bytes()[0] != latest {
			t.Error("maintenance rendered stale scene")
		}
		sent = append(sent, clock.tick)
		if len(sent) == 2 {
			cancel()
		}
		return nil
	})
	p, _ := NewScreenPump(s, sender, 700*time.Second)
	p.sleep = func(ctx context.Context, wake, changed <-chan struct{}, delay time.Duration, pending bool) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !pending {
			return errors.New("maintenance disappeared")
		}
		clock.advance(min(delay, 50*time.Second))
		latest++
		submitCycle(t, s, clock, string([]byte{latest}))
		return nil
	}
	if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if !slices.Equal(sent, []time.Duration{4300 * time.Second, 5000 * time.Second}) {
		t.Fatal("debounce starvation, cooldown bypass or catchup burst", sent)
	}
	if s.Status().FullRefresh.Cycle != 3 || s.Status().Confirmed != s.Status().Current {
		t.Fatal(s.Status())
	}
}
