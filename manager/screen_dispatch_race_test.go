package manager

import (
	"context"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func TestUnsentSupersessionAfterReadinessBeforeDispatch(t *testing.T) {
	r, c := unsentFixture(t)
	s := r.pump.screen
	// Simulate author replacement while WaitReady is blocked, after waitReady's
	// prior snapshot check. No old full frame may escape after reconnection.
	c.advance(time.Minute)
	submitCycle(t, s, c, "C")
	if err := r.deliver(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.unsent || s.Status().InFlight != 0 || s.Status().Confirmed != 1 {
		t.Fatal(s.Status(), r.unsent)
	}
}

func TestUnsentMaintenanceSupersededByValidAuthorWork(t *testing.T) {
	s, c := partialFixture(t)
	submitCycle(t, s, c, "A")
	confirmCycle(t, s, nextCycle(t, s, c))
	p := dispatchPump(t, s, c, &dispatchSender{onSend: func(refreshpolicy.Mode) error {
		t.Fatal("stale maintenance escaped")
		return nil
	}})
	r := screenRecovery{pump: p}
	c.advance(DefaultMaintenanceInterval)
	r.completed()
	if err := r.deliver(context.Background()); err != nil || !r.unsent {
		t.Fatal(err)
	}
	submitCycle(t, s, c, "B")
	c.advance(30 * time.Second)
	if err := r.deliver(context.Background()); err != nil || r.unsent {
		t.Fatal(err, r.unsent)
	}
	if s.Status().Confirmed != 1 || s.Status().Current != 2 {
		t.Fatal(s.Status())
	}
}

func TestAreaFallbackWaitsForFullAndStampsDispatch(t *testing.T) {
	for _, abort := range []bool{false, true} {
		t.Run(map[bool]string{false: "dispatch", true: "cancel"}[abort], func(t *testing.T) {
			testAreaFallback(t, abort)
		})
	}
}

func testAreaFallback(t *testing.T, abort bool) {
	t.Helper()
	s, c := partialFixture(t)
	wideDamageFixture(s)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sends, staged := 0, false
	p := dispatchPump(t, s, c, &dispatchSender{onSend: func(mode refreshpolicy.Mode) error {
		sends++
		if mode != refreshpolicy.Full || c.tick != time.Duration(sends)*30*time.Second {
			t.Fatal("early or wrong-mode dispatch", mode, c.tick)
		}
		if sends == 1 {
			submitCycle(t, s, c, "B")
		} else {
			cancel()
		}
		return nil
	}})
	sleep := p.sleep
	p.sleep = func(ctx context.Context, wake, changed <-chan struct{}, delay time.Duration, pending bool) error {
		if s.Status().InFlight != 0 {
			staged = true
			if abort {
				cancel()
			}
		}
		return sleep(ctx, wake, changed, delay, pending)
	}
	submitCycle(t, s, c, "A")
	if err := p.Run(ctx); !errors.Is(err, context.Canceled) || !staged {
		t.Fatal(err, staged)
	}
	checkAreaFallbackResult(t, s, c, abort, sends)
}

func wideDamageFixture(s *Screen) {
	renderer := s.cycles.renderer
	s.cycles.partialOptions.Rules.MaxBytes = 4
	s.cycles.renderer = reservedRenderer{render: func(ctx context.Context, size display.Size, html []byte, area image.Rectangle) (display.Frame, []renderdiag.Warning, error) {
		frame, warnings, err := renderer.RenderReserved(ctx, size, html, area)
		if err == nil && len(html) > 0 {
			frame.Bytes()[size.Width/8-1] = html[0]
		}
		return frame, warnings, err
	}}
}

func checkAreaFallbackResult(t *testing.T, s *Screen, c *cycleClock, abort bool, sends int) {
	t.Helper()
	want := 2
	if abort {
		want = 1
	}
	status := s.Status()
	if sends != want || int(status.Confirmed) != want || status.InFlight != 0 || !status.RefreshTrusted {
		t.Fatal(sends, status)
	}
	if !abort && !status.FullRefresh.Started.Equal(c.wall) {
		t.Fatal("full stamped before dispatch", status.FullRefresh, c.wall)
	}
	if _, err := s.cycles.tracker.ForPartial(); err != nil {
		t.Fatal(err)
	}
}
