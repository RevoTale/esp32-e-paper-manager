package manager

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func TestPartialLostACKRestoresTrustWithoutMovingFullHistory(t *testing.T) {
	s, c := partialFixture(t)
	submitCycle(t, s, c, "A")
	confirmCycle(t, s, nextCycle(t, s, c))
	proof := *s.Status().FullRefresh
	c.advance(time.Second)
	submitCycle(t, s, c, "B")
	d := nextCycle(t, s, c)
	r := screenRecovery{pump: &ScreenPump{screen: s}, pending: d}
	r.uncertain()
	if s.Status().RefreshTrusted {
		t.Fatal("unconfirmed partial retained trust")
	}
	if err := r.resolve(true); err != nil {
		t.Fatal(err)
	}
	if !s.Status().RefreshTrusted || *s.Status().FullRefresh != proof || s.Status().Confirmed != 2 {
		t.Fatal(s.Status())
	}
}

func TestPartialConfigurationAndFallback(t *testing.T) {
	s, c := partialFixture(t)
	if err := s.configurePartial(&ScreenPartialOptions{}); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	if err := s.configurePartial(&ScreenPartialOptions{MaxConsecutive: 1}); !errors.Is(err, screendelivery.ErrRegionInput) {
		t.Fatal(err)
	}
	if _, err := NewScreenPump(s, &partialSender{}, time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := NewScreenPump(s, noRegionSender{}, time.Second); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	submitCycle(t, s, c, "A")
	confirmCycle(t, s, nextCycle(t, s, c))
	c.advance(time.Second)
	submitCycle(t, s, c, "B")
	s.options.Mode = refreshpolicy.Partial
	s.cycles.partials = s.cycles.partialOptions.MaxConsecutive
	if _, err := s.RenderNext(context.Background(), c.tick, true); !errors.Is(err, refreshstamp.ErrUnconfirmed) {
		t.Fatal(err)
	}
	submitCycle(t, s, c, "C")
	s.cycles.partials = 0
	s.cycles.next = math.MaxUint64
	if _, err := s.RenderNext(context.Background(), c.tick, true); !errors.Is(err, refreshstamp.ErrCycle) {
		t.Fatal(err)
	}
}

type noRegionSender struct{}

func (noRegionSender) Send(context.Context, display.Frame) error { return nil }

type partialSender struct {
	noRegionSender
	regions int
}

func (p *partialSender) SendRegion(_ context.Context, r screendelivery.RegionPlan, o refreshpolicy.Options) error {
	if r.Bounds.Empty() || o.Mode != refreshpolicy.Partial {
		return ErrConfiguration
	}
	p.regions++
	return nil
}

func TestPartialRecoveryDispatchAndClockGuard(t *testing.T) {
	s, c := partialFixture(t)
	submitCycle(t, s, c, "A")
	confirmCycle(t, s, nextCycle(t, s, c))
	c.advance(time.Second)
	submitCycle(t, s, c, "B")
	d := nextCycle(t, s, c)
	sender := &partialSender{}
	r := screenRecovery{pump: &ScreenPump{screen: s, sender: sender, interval: time.Second}, pending: d}
	if err := r.send(context.Background()); err != nil || sender.regions != 1 {
		t.Fatal(err, sender)
	}
	c.advance(time.Second)
	submitCycle(t, s, c, "C")
	d = nextCycle(t, s, c)
	c.tick = 0
	if err := s.ResolveCycle(d.Cycle, true); !errors.Is(err, renderbatch.ErrClock) {
		t.Fatal(err)
	}
	if s.Status().RefreshTrusted || s.Status().Confirmed != 0 {
		t.Fatal(s.Status())
	}
}
