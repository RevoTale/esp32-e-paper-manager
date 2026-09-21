package manager

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
)

func requireScreen(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestScreenInputAndPolicyBounds(t *testing.T) {
	r := screenRenderer(blackRenderer)
	for _, size := range []display.Size{{}, {Width: 2049, Height: 1}, {Width: 2048, Height: 2048}} {
		if _, err := NewScreen(r, size, renderbatch.Policy{}); err == nil {
			t.Fatal("size")
		}
	}
	if _, err := NewScreen(nil, display.Size{Width: 8, Height: 1}, renderbatch.Policy{}); err == nil {
		t.Fatal("renderer")
	}
	if _, err := NewScreen(r, display.Size{Width: 8, Height: 1}, renderbatch.Policy{Debounce: -1}); err == nil {
		t.Fatal("policy")
	}
	s := screenFixture(t, r)
	for _, input := range [][]byte{{255}, make([]byte, 32769)} {
		if _, err := s.Submit(0, input, 0); err == nil {
			t.Fatal("input")
		}
	}
	if _, err := s.Submit(0, nil, -1); err == nil {
		t.Fatal("clock")
	}
	s.state.Current = math.MaxUint64
	if _, err := s.Submit(math.MaxUint64, nil, 0); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}

func TestScreenEmptyUnavailableAndFailure(t *testing.T) {
	s := screenFixture(t, screenRenderer(blackRenderer))
	if d, err := s.RenderNext(context.Background(), 0, true); err != nil || d.Revision != 0 {
		t.Fatal(d, err)
	}
	_, err := s.Submit(0, []byte("one"), 0)
	requireScreen(t, err)
	if d, err := s.RenderNext(context.Background(), 0, false); err != nil || d.Revision != 0 {
		t.Fatal(d, err)
	}
	d, err := s.RenderNext(context.Background(), 0, true)
	requireScreen(t, err)
	requireScreen(t, s.Resolve(d.Revision, true))
	// A changed target is required to exercise failed delivery; equal pixels
	// are now confirmed by equivalence without another hardware operation.
	s.renderer = apiRenderer{}
	_, err = s.Submit(d.Revision, []byte("two"), 0)
	requireScreen(t, err)
	d, err = s.RenderNext(context.Background(), 0, true)
	requireScreen(t, err)
	requireScreen(t, s.Resolve(d.Revision, false))
	if s.Status().Confirmed != 0 {
		t.Fatal("unknown result retained baseline")
	}
}

func TestScreenOwnsMarkupAndRejectsBadFrames(t *testing.T) {
	var observed string
	s := screenFixture(t, screenRenderer(func(ctx context.Context, size display.Size, p []byte) (display.Frame, error) {
		observed = string(p)
		return blackRenderer(ctx, size, p)
	}))
	input := []byte("old")
	_, err := s.Submit(0, input, 0)
	requireScreen(t, err)
	input[0] = 'X'
	_, err = s.RenderNext(context.Background(), 0, true)
	requireScreen(t, err)
	if observed != "old" {
		t.Fatal("borrowed mutable input")
	}
	for _, renderer := range []ScreenRenderer{
		screenRenderer(func(context.Context, display.Size, []byte) (display.Frame, error) {
			return display.Frame{}, errors.New("render failed")
		}),
		screenRenderer(func(context.Context, display.Size, []byte) (display.Frame, error) { return display.Frame{}, nil }),
	} {
		bad := screenFixture(t, renderer)
		_, err = bad.Submit(0, nil, 0)
		requireScreen(t, err)
		if _, err = bad.RenderNext(context.Background(), 0, true); err == nil {
			t.Fatal("bad render")
		}
		if bad.Status().InFlight != 0 {
			t.Fatal("lease leaked")
		}
	}
}

func TestScreenPaddingAndPrematureResolution(t *testing.T) {
	s, err := NewScreen(screenRenderer(blackRenderer), display.Size{Width: 7, Height: 1}, renderbatch.Policy{})
	requireScreen(t, err)
	_, err = s.Submit(0, nil, 0)
	requireScreen(t, err)
	if err = s.Resolve(1, true); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if _, err = s.RenderNext(context.Background(), 0, true); !errors.Is(err, display.ErrFrameGeometry) {
		t.Fatal(err)
	}
	f, err := display.NewFrame(display.Size{Width: 7, Height: 1}, 1, []byte{254})
	requireScreen(t, err)
	if !s.validFrame(f) {
		t.Fatal("valid padding")
	}
}
