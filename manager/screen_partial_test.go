package manager

import (
	"bytes"
	"context"
	"image"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func partialFixture(t *testing.T) (*Screen, *cycleClock) {
	t.Helper()
	c := &cycleClock{wall: time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)}
	renderer := reservedRenderer{render: func(_ context.Context, size display.Size, html []byte, _ image.Rectangle) (display.Frame, []renderdiag.Warning, error) {
		pixels := make([]byte, (size.Width/8)*size.Height)
		if len(html) > 0 {
			pixels[0] = html[0]
		}
		frame, err := display.NewFrame(size, size.Width/8, pixels)
		return frame, nil, err
	}}
	s, err := NewScreenWithOptions(renderer, display.Size{Width: 200, Height: 40}, renderbatch.Policy{}, ScreenOptions{
		Zone: time.UTC, Now: func() time.Time { return c.wall },
		Partial: &ScreenPartialOptions{Rules: screendelivery.RegionRules{MinWidth: 16, MinHeight: 2, MaxBytes: 100}, MaxConsecutive: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.elapsedClock = func() time.Duration { return c.tick }
	return s, c
}

func TestPartialPreservesFullTimestampAndMaintenance(t *testing.T) {
	s, c := partialFixture(t)
	submitCycle(t, s, c, "A")
	full := nextCycle(t, s, c)
	confirmCycle(t, s, full)
	proof := *s.Status().FullRefresh
	c.advance(time.Minute)
	submitCycle(t, s, c, "B")
	part := nextCycle(t, s, c)
	if part.Region == nil || part.Options.Mode != refreshpolicy.Partial || part.Cycle == full.Cycle {
		t.Fatal(part)
	}
	assertPartialReplay(t, full.Frame, part)
	confirmCycle(t, s, part)
	if *s.Status().FullRefresh != proof || !s.Status().RefreshTrusted || s.Status().Confirmed != 2 {
		t.Fatal(s.Status())
	}
	c.advance(9 * time.Minute)
	maintenance := nextCycle(t, s, c)
	if maintenance.Region != nil || maintenance.Options.Mode != refreshpolicy.Full {
		t.Fatal(maintenance)
	}
	confirmCycle(t, s, maintenance)
}

func assertPartialReplay(t *testing.T, base display.Frame, d ScreenDelivery) {
	t.Helper()
	r := d.Region
	if r == nil {
		t.Fatal("missing region")
	}
	out := bytes.Clone(base.Bytes())
	stride := r.Bounds.Dx() / 8
	for y := r.Bounds.Min.Y; y < r.Bounds.Max.Y; y++ {
		start := y*base.Stride() + r.Bounds.Min.X/8
		row := (y - r.Bounds.Min.Y) * stride
		if !bytes.Equal(out[start:start+stride], r.Old[row:row+stride]) {
			t.Fatal("wrong confirmed pixels")
		}
		copy(out[start:start+stride], r.New[row:row+stride])
	}
	if !bytes.Equal(out, d.Frame.Bytes()) {
		t.Fatal("partial replay differs from target")
	}
	area, _ := refreshstamp.Bounds(base.Size())
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			if base.Pixel(x, y) != d.Frame.Pixel(x, y) {
				t.Fatal("partial changed timestamp")
			}
		}
	}
}

func TestPartialFailureInvalidatesBase(t *testing.T) {
	s, c := partialFixture(t)
	submitCycle(t, s, c, "A")
	confirmCycle(t, s, nextCycle(t, s, c))
	c.advance(time.Second)
	submitCycle(t, s, c, "B")
	d := nextCycle(t, s, c)
	if d.Region == nil {
		t.Fatal("missing partial")
	}
	if err := s.ResolveCycle(d.Cycle, false); err != nil {
		t.Fatal(err)
	}
	if s.Status().RefreshTrusted || s.Status().Confirmed != 0 {
		t.Fatal(s.Status())
	}
	submitCycle(t, s, c, "C")
	d = nextCycle(t, s, c)
	if d.Region != nil || d.Options.Mode != refreshpolicy.Full {
		t.Fatal("missing full resync", d)
	}
}

func TestPartialCountForcesFullCycle(t *testing.T) {
	s, c := partialFixture(t)
	for i, html := range []string{"A", "B", "C", "D"} {
		submitCycle(t, s, c, html)
		d := nextCycle(t, s, c)
		if (d.Region != nil) != (i == 1 || i == 2) {
			t.Fatal(i, d)
		}
		confirmCycle(t, s, d)
		c.advance(time.Second)
	}
}

func TestExplicitFullRefreshDoesNotSuppressIdenticalPixels(t *testing.T) {
	s, c := partialFixture(t)
	submitCycle(t, s, c, "A")
	confirmCycle(t, s, nextCycle(t, s, c))
	c.advance(time.Minute)
	submitCycle(t, s, c, "A")
	s.options.Mode = refreshpolicy.Full
	d := nextCycle(t, s, c)
	if d.Revision == 0 || d.Region != nil || d.Options.Mode != refreshpolicy.Full {
		t.Fatal(d)
	}
}
