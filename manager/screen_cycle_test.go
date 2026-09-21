package manager

import (
	"bytes"
	"context"
	"image"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

type cycleClock struct {
	wall time.Time
	tick time.Duration
}

func (c *cycleClock) advance(d time.Duration) { c.wall = c.wall.Add(d); c.tick += d }

type reservedRenderer struct {
	render func(context.Context, display.Size, []byte, image.Rectangle) (display.Frame, []renderdiag.Warning, error)
}

func (r reservedRenderer) Render(ctx context.Context, size display.Size, html []byte) (display.Frame, error) {
	f, _, err := r.RenderReserved(ctx, size, html, image.Rectangle{})
	return f, err
}

func (r reservedRenderer) RenderReserved(ctx context.Context, size display.Size, html []byte, area image.Rectangle) (display.Frame, []renderdiag.Warning, error) {
	if r.render != nil {
		return r.render(ctx, size, html, area)
	}
	stride := (size.Width + 7) / 8
	f, err := display.NewFrame(size, stride, make([]byte, stride*size.Height))
	return f, nil, err
}

func cycleFixture(t *testing.T, policy renderbatch.Policy) (*Screen, *cycleClock) {
	t.Helper()
	c := &cycleClock{wall: time.Date(2026, 9, 7, 12, 34, 0, 0, time.UTC)}
	s, err := NewScreenWithOptions(reservedRenderer{}, display.Size{Width: 200, Height: 40}, policy,
		ScreenOptions{Zone: time.FixedZone("fixture", 2*60*60), Now: func() time.Time { return c.wall }})
	if err != nil {
		t.Fatal(err)
	}
	s.elapsedClock = func() time.Duration { return c.tick }
	return s, c
}

func submitCycle(t *testing.T, s *Screen, c *cycleClock, html string) {
	t.Helper()
	if _, err := s.Submit(s.Status().Current, []byte(html), c.tick); err != nil {
		t.Fatal(err)
	}
}

func nextCycle(t *testing.T, s *Screen, c *cycleClock) ScreenDelivery {
	t.Helper()
	d, err := s.RenderNext(context.Background(), c.tick, true)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func confirmCycle(t *testing.T, s *Screen, d ScreenDelivery) {
	t.Helper()
	if err := s.ResolveCycle(d.Cycle, true); err != nil {
		t.Fatal(err)
	}
}

func TestScreenCycleStampUsesTimezoneAndAwaitsACK(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	submitCycle(t, s, c, "first")
	first := nextCycle(t, s, c)
	if first.Cycle != 1 || first.Revision != 1 || s.Status().FullRefresh != nil {
		t.Fatal(first, s.Status())
	}
	tracker, _ := refreshstamp.New(time.FixedZone("fixture", 2*60*60))
	label, _ := tracker.Begin(1, c.wall)
	expected, _, _ := (reservedRenderer{}).RenderReserved(context.Background(), s.size, nil, image.Rectangle{})
	if err := refreshstamp.Paint(expected, label); err != nil || !bytes.Equal(first.Frame.Bytes(), expected.Bytes()) {
		t.Fatal("wrong timezone stamp", err)
	}
	confirmCycle(t, s, first)
}

func TestScreenCycleNoOpDoesNotAdvancePhysicalHistory(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	submitCycle(t, s, c, "first")
	first := nextCycle(t, s, c)
	c.advance(10 * time.Second)
	confirmCycle(t, s, first)
	confirmed := *s.Status().FullRefresh
	c.advance(589 * time.Second)
	submitCycle(t, s, c, "pixel-equivalent")
	if d := nextCycle(t, s, c); d.Revision != 0 || s.Status().Confirmed != 2 || s.Status().Delivered != 1 {
		t.Fatal(d, s.Status())
	}
	if d := nextCycle(t, s, c); d.Revision != 0 {
		t.Fatal("early maintenance", d)
	}
	if *s.Status().FullRefresh != confirmed {
		t.Fatal("no-op advanced physical proof", s.Status())
	}
}

func TestScreenCycleMaintenanceDeadlineUsesCompletion(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	submitCycle(t, s, c, "first")
	first := nextCycle(t, s, c)
	c.advance(10 * time.Second)
	confirmCycle(t, s, first)
	confirmed := *s.Status().FullRefresh
	c.advance(DefaultMaintenanceInterval - 1)
	if d := nextCycle(t, s, c); d.Revision != 0 {
		t.Fatal("early maintenance", d)
	}
	c.advance(1)
	maintenance := nextCycle(t, s, c)
	if maintenance.Cycle != 2 || maintenance.Revision != 1 || *s.Status().FullRefresh != confirmed {
		t.Fatal(maintenance, s.Status())
	}
	confirmCycle(t, s, maintenance)
	if s.Status().FullRefresh.Cycle != 2 || !s.Status().RefreshTrusted || s.Status().Current != 1 {
		t.Fatal(s.Status())
	}
	if d := nextCycle(t, s, c); d.Revision != 0 {
		t.Fatal("catch-up burst", d)
	}
}
