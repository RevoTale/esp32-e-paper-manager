package manager

import (
	"context"
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
)

type screenRenderer func(context.Context, display.Size, []byte) (display.Frame, error)

func (f screenRenderer) Render(ctx context.Context, size display.Size, html []byte) (display.Frame, error) {
	return f(ctx, size, html)
}

func screenFixture(t *testing.T, renderer ScreenRenderer) *Screen {
	t.Helper()
	s, err := NewScreen(renderer, display.Size{Width: 8, Height: 1}, renderbatch.Policy{})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func blackRenderer(_ context.Context, size display.Size, _ []byte) (display.Frame, error) {
	return display.NewFrame(size, 1, []byte{255})
}

func TestScreenRejectsStaleEditsAndACK(t *testing.T) {
	s := screenFixture(t, screenRenderer(blackRenderer))
	rev, err := s.Submit(0, []byte("one"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Submit(0, []byte("stale"), 0); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	d, err := s.RenderNext(context.Background(), 0, true)
	if err != nil || d.Revision != rev {
		t.Fatal(d, err)
	}
	if err = s.Resolve(rev+1, true); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if err = s.Resolve(rev, true); err != nil {
		t.Fatal(err)
	}
	if s.Status().Confirmed != rev {
		t.Fatal(s.Status())
	}
}

func TestScreenDiscardsRenderSupersededByEdit(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	s := screenFixture(t, screenRenderer(func(ctx context.Context, size display.Size, html []byte) (display.Frame, error) {
		close(started)
		<-release
		return blackRenderer(ctx, size, html)
	}))
	if _, err := s.Submit(0, []byte("old"), 0); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := s.RenderNext(context.Background(), 0, true); done <- err }()
	<-started
	if _, err := s.Submit(1, []byte("new"), 1); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-done; !errors.Is(err, ErrSuperseded) {
		t.Fatal(err)
	}
	if s.Status().Confirmed != 0 || s.Status().InFlight != 0 {
		t.Fatal(s.Status())
	}
}
