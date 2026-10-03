package manager

import (
	"context"
	"errors"
	"image"
	"math"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func TestScreenCycleOptionsRejectUnsafeConfigurations(t *testing.T) {
	base := display.Size{Width: 200, Height: 40}
	cases := []struct {
		name     string
		renderer ScreenRenderer
		size     display.Size
		policy   renderbatch.Policy
		options  ScreenOptions
	}{
		{"nil renderer", nil, base, renderbatch.Policy{}, ScreenOptions{Zone: time.UTC}},
		{"bad size", reservedRenderer{}, display.Size{}, renderbatch.Policy{}, ScreenOptions{Zone: time.UTC}},
		{"bad batching", reservedRenderer{}, base, renderbatch.Policy{Debounce: -1}, ScreenOptions{Zone: time.UTC}},
		{"no reservation", screenRenderer(blackRenderer), base, renderbatch.Policy{}, ScreenOptions{Zone: time.UTC}},
		{"negative maintenance", reservedRenderer{}, base, renderbatch.Policy{}, ScreenOptions{Zone: time.UTC, MaintenanceInterval: -1}},
		{"missing zone", reservedRenderer{}, base, renderbatch.Policy{}, ScreenOptions{}},
		{"small display", reservedRenderer{}, display.Size{Width: 8, Height: 1}, renderbatch.Policy{}, ScreenOptions{Zone: time.UTC}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewScreenWithOptions(tt.renderer, tt.size, tt.policy, tt.options); err == nil {
				t.Fatal("accepted unsafe configuration")
			}
		})
	}
	s, err := NewScreenWithOptions(reservedRenderer{}, base, renderbatch.Policy{}, ScreenOptions{Zone: time.UTC, MaintenanceInterval: time.Minute})
	if err != nil || s.cycles.interval != time.Minute || s.cycles.now().IsZero() {
		t.Fatal(s, err)
	}
}

func TestScreenCycleRejectsExhaustionAndOccupiedReservation(t *testing.T) {
	for _, exhausted := range []bool{false, true} {
		t.Run(map[bool]string{false: "occupied", true: "exhausted"}[exhausted], func(t *testing.T) {
			s, c := cycleFixture(t, renderbatch.Policy{})
			expected := refreshstamp.ErrOccupied
			if exhausted {
				s.cycles.next = math.MaxUint64
				expected = refreshstamp.ErrCycle
			} else {
				s.cycles.renderer = occupiedRenderer()
			}
			submitCycle(t, s, c, "one")
			if d, err := s.RenderNext(context.Background(), c.tick, true); !errors.Is(err, expected) || d.Cycle != 0 {
				t.Fatal(d, err)
			}
			if s.Status().InFlight != 0 || s.Status().FullRefresh != nil || s.Status().RefreshTrusted {
				t.Fatal(s.Status())
			}
		})
	}
}

func occupiedRenderer() reservedRenderer {
	return reservedRenderer{render: func(ctx context.Context, size display.Size, html []byte, area image.Rectangle) (display.Frame, []renderdiag.Warning, error) {
		f, warnings, err := (reservedRenderer{}).RenderReserved(ctx, size, html, area)
		f.Bytes()[area.Min.Y*f.Stride()+area.Min.X/8] |= 0x80 >> uint(area.Min.X%8)
		return f, warnings, err
	}}
}

func TestScreenCycleLeaseRejectsConcurrentRenderAndInvalidation(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	submitCycle(t, s, c, "one")
	d := nextCycle(t, s, c)
	if err := s.InvalidatePixels(); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if err := s.forceLatest(); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if next := nextCycle(t, s, c); next.Revision != 0 {
		t.Fatal("second concurrent cycle", next)
	}
	confirmCycle(t, s, d)
}

func TestScreenCycleInvalidationAndResyncKeepAuthorRevision(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	submitCycle(t, s, c, "one")
	confirmCycle(t, s, nextCycle(t, s, c))
	if err := s.InvalidatePixels(); err != nil || s.Status().RefreshTrusted || s.Status().FullRefresh.Cycle != 1 {
		t.Fatal(err, s.Status())
	}
	if err := s.forceLatest(); err != nil {
		t.Fatal(err)
	}
	d := nextCycle(t, s, c)
	if d.Cycle != 2 || d.Revision != 1 || s.Status().Current != 1 {
		t.Fatal(d, s.Status())
	}
	confirmCycle(t, s, d)
}

func TestScreenCycleCannotResolveBeforeRenderingFinishes(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	s.cycles.renderer = reservedRenderer{render: func(ctx context.Context, size display.Size, html []byte, area image.Rectangle) (display.Frame, []renderdiag.Warning, error) {
		if err := s.ResolveCycle(1, true); !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
		return markRenderer(ctx, size, html, area)
	}}
	submitCycle(t, s, c, "one")
	confirmCycle(t, s, nextCycle(t, s, c))
}

func TestScreenCycleResyncWithoutSceneIsIdle(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	if err := s.forceLatest(); err != nil {
		t.Fatal(err)
	}
	if d := nextCycle(t, s, c); d.Cycle != 0 || s.cycles.resync {
		t.Fatal(d, s.cycles.resync)
	}
	if _, pending, err := s.wakeDelay(c.tick); err != nil || pending {
		t.Fatal(pending, err)
	}
	legacy := screenFixture(t, screenRenderer(blackRenderer))
	if err := legacy.ResolveCycle(1, true); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if err := legacy.forceLatest(); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}

func TestScreenCycleBareRejectionIsFatalWithoutInventingDiagnostic(t *testing.T) {
	s, c := cycleFixture(t, renderbatch.Policy{})
	s.cycles.renderer = reservedRenderer{render: func(context.Context, display.Size, []byte, image.Rectangle) (display.Frame, []renderdiag.Warning, error) {
		return display.Frame{}, nil, renderdiag.ErrRejected
	}}
	submitCycle(t, s, c, "one")
	_, err := s.RenderNext(context.Background(), c.tick, true)
	if !errors.Is(err, ErrConfiguration) || errors.Is(err, renderdiag.ErrRejected) {
		t.Fatal("unclassified rejection would retry forced maintenance", err)
	}
	if s.Status().InFlight != 0 || s.Status().Failure != nil || s.cycles.rejected != 0 {
		t.Fatal(s.Status(), s.cycles.rejected)
	}
}
