package manager

import (
	"context"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func markRenderer(ctx context.Context, size display.Size, html []byte, area image.Rectangle) (display.Frame, []renderdiag.Warning, error) {
	f, warnings, err := (reservedRenderer{}).RenderReserved(ctx, size, html, area)
	if len(html) != 0 {
		f.Bytes()[0] = html[0]
	}
	return f, warnings, err
}

func TestMaintenanceTakesLatestBeforeDebounceThenQueueNoOp(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{Debounce: time.Minute})
	s.cycles.renderer = reservedRenderer{render: markRenderer}
	submitCycle(t, s, c, "A")
	c.advance(time.Minute)
	confirmCycle(t, s, nextCycle(t, s, c))
	c.advance(599 * time.Second)
	submitCycle(t, s, c, "B")
	c.advance(time.Second)
	submitCycle(t, s, c, "C")
	d := nextCycle(t, s, c)
	if d.Revision != 3 || d.Cycle != 2 || d.Frame.Bytes()[0] != 'C' {
		t.Fatal("maintenance sent stale content", d)
	}
	confirmCycle(t, s, d)
	c.advance(time.Minute)
	if next := nextCycle(t, s, c); next.Revision != 0 || s.Status().Current != 3 || s.Status().Confirmed != 3 {
		t.Fatal("later queue entry was not a no-op", next, s.Status())
	}
}

func TestMaintenanceDiscardSupersededForcedRender(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	submitCycle(t, s, c, "A")
	confirmCycle(t, s, nextCycle(t, s, c))
	c.advance(DefaultMaintenanceInterval)
	submitCycle(t, s, c, "B")
	s.cycles.renderer = reservedRenderer{render: func(ctx context.Context, size display.Size, html []byte, area image.Rectangle) (display.Frame, []renderdiag.Warning, error) {
		submitCycle(t, s, c, "C")
		return markRenderer(ctx, size, html, area)
	}}
	if d, err := s.RenderNext(context.Background(), c.tick, true); !errors.Is(err, ErrSuperseded) || d.Revision != 0 {
		t.Fatal(d, err)
	}
	if s.Status().InFlight != 0 || s.Status().FullRefresh.Cycle != 1 {
		t.Fatal(s.Status())
	}
	s.cycles.renderer = reservedRenderer{render: markRenderer}
	d := nextCycle(t, s, c)
	if d.Revision != 3 || d.Frame.Bytes()[0] != 'C' || d.Cycle != 2 {
		t.Fatal(d)
	}
	confirmCycle(t, s, d)
}

func TestRejectedLatestMaintenanceKeepsOlderRevisionAndDiagnostic(t *testing.T) {
	s, c, calls := rejectedMaintenance(t)
	maintenance := nextCycle(t, s, c)
	if maintenance.Revision != 1 || maintenance.Cycle != 2 || s.Status().Failure == nil {
		t.Fatal(maintenance, s.Status())
	}
	confirmCycle(t, s, maintenance)
	c.advance(time.Minute)
	if d := nextCycle(t, s, c); d.Revision != 0 || *calls != 1 {
		t.Fatal("repeated rejected render", d, *calls)
	}
	status := s.Status()
	if status.Current != 2 || status.Confirmed != 1 || status.Delivered != 1 || status.Failure.Revision != 2 {
		t.Fatal(status)
	}
}

func TestRejectedMaintenanceAllowsExplicitCorrection(t *testing.T) {
	s, c, _ := rejectedMaintenance(t)
	confirmCycle(t, s, nextCycle(t, s, c))
	s.cycles.renderer = reservedRenderer{}
	submitCycle(t, s, c, "valid again")
	c.advance(time.Minute)
	if d := nextCycle(t, s, c); d.Revision != 0 || s.Status().Confirmed != 3 || s.Status().Failure != nil {
		t.Fatal(d, s.Status())
	}
}

func rejectedMaintenance(t *testing.T) (*Screen, *cycleClock, *int) {
	t.Helper()
	s, c := cycleFixture(t, renderbatch.Policy{Debounce: time.Minute})
	submitCycle(t, s, c, "good")
	c.advance(time.Minute)
	confirmCycle(t, s, nextCycle(t, s, c))
	c.advance(DefaultMaintenanceInterval)
	submitCycle(t, s, c, "bad")
	calls := 0
	s.cycles.renderer = reservedRenderer{render: func(context.Context, display.Size, []byte, image.Rectangle) (display.Frame, []renderdiag.Warning, error) {
		calls++
		return display.Frame{}, nil, &renderdiag.Error{Code: renderdiag.InvalidValue}
	}}
	if _, err := s.RenderNext(context.Background(), c.tick, true); !errors.Is(err, renderdiag.ErrRejected) {
		t.Fatal(err)
	}
	return s, c, &calls
}

func TestMaintenanceUnavailableAndLongOfflineGapDoNotBurst(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	submitCycle(t, s, c, "first")
	confirmCycle(t, s, nextCycle(t, s, c))
	c.advance(24 * time.Hour)
	if d, err := s.RenderNext(context.Background(), c.tick, false); err != nil || d.Revision != 0 {
		t.Fatal(d, err)
	}
	d := nextCycle(t, s, c)
	if d.Cycle != 2 || d.Revision != 1 {
		t.Fatal(d)
	}
	c.advance(time.Minute)
	confirmCycle(t, s, d)
	delay, pending, err := s.wakeDelay(c.tick)
	if err != nil || !pending || delay != DefaultMaintenanceInterval {
		t.Fatal(delay, pending, err)
	}
	if next := nextCycle(t, s, c); next.Revision != 0 {
		t.Fatal(next)
	}
}
