package manager

import (
	"context"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func TestScreenStampRealCompositionPreservesViewportAndWarnings(t *testing.T) {
	renderer, err := engine.New()
	if err != nil {
		t.Fatal(err)
	}
	size := display.Size{Width: 200, Height: 80}
	c := &cycleClock{wall: time.Date(2026, 9, 7, 10, 20, 0, 0, time.UTC)}
	s, err := NewScreenWithOptions(renderer, size, renderbatch.Policy{}, ScreenOptions{Zone: time.UTC, Now: func() time.Time { return c.wall }})
	if err != nil {
		t.Fatal(err)
	}
	s.elapsedClock = func() time.Duration { return c.tick }
	submitCycle(t, s, c, `<div style="width:100vw;height:100vh;background:black"></div>`)
	d := nextCycle(t, s, c)
	area, _ := refreshstamp.Bounds(size)
	assertPixel(t, d.Frame, area.Min.X-1, size.Height-1, display.Black)
	assertPixel(t, d.Frame, size.Width-1, area.Min.Y-1, display.Black)
	assertPixel(t, d.Frame, area.Min.X, area.Min.Y, display.Black)
	assertPixel(t, d.Frame, area.Min.X+1, area.Min.Y+1, display.White)
	assertPixel(t, s.candidate, area.Min.X, area.Min.Y, display.White)
	if warnings := s.Status().Warnings; len(warnings) != 1 || warnings[0].Code != renderdiag.ReservedOverlap {
		t.Fatal(warnings)
	}
	confirmCycle(t, s, d)
	c.advance(time.Minute)
	submitCycle(t, s, c, `<div style="width:100vw;height:100vh;background:#000"></div>`)
	if next := nextCycle(t, s, c); next.Cycle != 0 || len(s.Status().Warnings) != 1 {
		t.Fatal("timestamp or warning broke no-op", next, s.Status())
	}
	// The public adapter also preserves its existing ordinary unreserved path.
	plain, err := renderer.Render(context.Background(), size, []byte(`<div style="width:100vw;height:100vh;background:black"></div>`))
	if err != nil {
		t.Fatal(err)
	}
	assertPixel(t, plain, area.Min.X+1, area.Min.Y+1, display.Black)
}

func assertPixel(t *testing.T, frame display.Frame, x, y int, expected display.Color) {
	t.Helper()
	if pixel := frame.Pixel(x, y); pixel != expected {
		t.Fatalf("pixel (%d,%d)=%v want %v", x, y, pixel, expected)
	}
}
