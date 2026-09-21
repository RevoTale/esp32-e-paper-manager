package manager

import (
	"context"
	"errors"
	"testing"
)

func nextScene(t *testing.T, s *Screen, markup string) ScreenDelivery {
	t.Helper()
	if _, err := s.Submit(s.Status().Current, []byte(markup), 0); err != nil {
		t.Fatal(err)
	}
	d, err := s.RenderNext(context.Background(), 0, true)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestConfirmedPixelsSuppressTransport(t *testing.T) {
	s := screenFixture(t, screenRenderer(blackRenderer))
	first := nextScene(t, s, "one")
	if first.Revision != 1 {
		t.Fatal(first)
	}
	if err := s.Resolve(first.Revision, true); err != nil {
		t.Fatal(err)
	}
	// The renderer creates independently owned output. Even after its original
	// delivery storage is reused, the confirmed baseline must remain immutable.
	first.Frame.Bytes()[0] = 0
	second := nextScene(t, s, "different HTML, same pixels")
	if second.Revision != 0 || s.Status().Confirmed != 2 || s.Status().Delivered != 1 || s.Status().InFlight != 0 {
		t.Fatal(second, s.Status())
	}
	if err := s.Resolve(2, true); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}

func TestUnknownPixelsRequireDelivery(t *testing.T) {
	s := screenFixture(t, screenRenderer(blackRenderer))
	first := nextScene(t, s, "one")
	if err := s.InvalidatePixels(); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if err := s.Resolve(first.Revision, false); err != nil {
		t.Fatal(err)
	}
	second := nextScene(t, s, "same")
	if second.Revision != 2 {
		t.Fatal(second)
	}
	if err := s.Resolve(second.Revision, true); err != nil {
		t.Fatal(err)
	}
	if err := s.InvalidatePixels(); err != nil {
		t.Fatal(err)
	}
	third := nextScene(t, s, "same again")
	if third.Revision != 3 || s.Status().Confirmed != 0 {
		t.Fatal(third, s.Status())
	}
}
