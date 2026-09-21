package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func TestScreenPumpOwnershipIsPerScreen(t *testing.T) {
	a, s := newScreenAPI(t)
	entered, release := make(chan struct{}), make(chan struct{})
	failure := errors.New("ambiguous")
	sender := screenSender(func(context.Context, display.Frame) error { close(entered); <-release; return failure })
	first, _ := NewScreenPump(s, sender, time.Nanosecond)
	second, _ := NewScreenPump(s, sender, time.Nanosecond)
	_ = apiRequest(a, "PUT", "/v2/screen", a.tag(0), "one")
	done := make(chan error, 1)
	go func() { done <- first.Run(context.Background()) }()
	<-entered
	if err := second.Run(context.Background()); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	_ = apiRequest(a, "PUT", "/v2/screen", a.tag(1), "pending")
	close(release)
	if err := <-done; !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if s.Status().Confirmed != 0 || s.Status().Current != 2 {
		t.Fatal(s.Status())
	}
}

func TestScreenPumpSupersededRenderAndFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		a, s := newScreenAPI(t)
		ctx, cancel := context.WithCancel(context.Background())
		failure := errors.New("render error")
		calls := 0
		s.renderer = screenRenderer(func(context.Context, display.Size, []byte) (display.Frame, error) {
			calls++
			if fail {
				return display.Frame{}, failure
			}
			if calls == 1 {
				_ = apiRequest(a, "PUT", "/v2/screen", a.tag(1), "two")
			}
			return (apiRenderer{}).Render(ctx, display.Size{}, nil)
		})
		p, _ := NewScreenPump(s, screenSender(func(context.Context, display.Frame) error { cancel(); return nil }), time.Nanosecond)
		_ = apiRequest(a, "PUT", "/v2/screen", a.tag(0), "one")
		err := p.Run(ctx)
		cancel()
		if fail && !errors.Is(err, failure) {
			t.Fatal(err)
		}
		if !fail && (calls != 2 || s.Status().Confirmed != 2) {
			t.Fatal(calls, s.Status())
		}
	}
}

func TestScreenPumpConfigurationAndWait(t *testing.T) {
	_, s := newScreenAPI(t)
	sender := screenSender(func(context.Context, display.Frame) error { return nil })
	for _, tc := range []struct {
		s        *Screen
		sender   ScreenSender
		interval time.Duration
	}{
		{nil, sender, time.Second}, {s, nil, time.Second}, {s, sender, 0}, {s, sender, -1},
	} {
		if _, err := NewScreenPump(tc.s, tc.sender, tc.interval); !errors.Is(err, ErrConfiguration) {
			t.Fatal(err)
		}
	}
	if err := waitScreen(context.Background(), nil, time.Nanosecond, true); err != nil {
		t.Fatal(err)
	}
	wake := make(chan struct{}, 1)
	wake <- struct{}{}
	if err := waitScreen(context.Background(), wake, 0, false); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitScreen(ctx, nil, 0, false); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
