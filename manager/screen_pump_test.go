package manager

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

type screenSender func(context.Context, display.Frame) error

func (f screenSender) Send(ctx context.Context, frame display.Frame) error { return f(ctx, frame) }

func TestScreenPumpAPIToDelivery(t *testing.T) {
	a, s := newScreenAPI(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var sent atomic.Int32
	p, err := NewScreenPump(s, screenSender(func(context.Context, display.Frame) error { sent.Add(1); cancel(); return nil }), time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	if w := apiRequest(a, "PUT", "/v2/screen", a.tag(0), "<b>screen</b>"); w.Code != 202 {
		t.Fatal(w)
	}
	if err = p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if sent.Load() != 1 || s.Status().Confirmed != 1 || s.Status().InFlight != 0 {
		t.Fatal(sent.Load(), s.Status())
	}
}

func TestScreenPumpStopsOnUnknownResult(t *testing.T) {
	a, s := newScreenAPI(t)
	failure := errors.New("lost commit ACK")
	count := 0
	p, err := NewScreenPump(s, screenSender(func(context.Context, display.Frame) error {
		count++
		if w := apiRequest(a, "PUT", "/v2/screen", a.tag(1), "newest"); w.Code != 202 {
			t.Fatal(w)
		}
		return failure
	}), time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	_ = apiRequest(a, "PUT", "/v2/screen", a.tag(0), "first")
	if err = p.Run(context.Background()); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if count != 1 || s.Status().Confirmed != 0 || s.Status().Current != 2 || s.Status().InFlight != 0 {
		t.Fatal(count, s.Status())
	}
}

func TestScreenPumpWaitsWithoutWorkAndStartupCooldown(t *testing.T) {
	for _, submit := range []bool{false, true} {
		a, s := newScreenAPI(t)
		if submit {
			_ = apiRequest(a, "PUT", "/v2/screen", a.tag(0), "first")
		}
		p, err := NewScreenPump(s, screenSender(func(context.Context, display.Frame) error { t.Error("unexpected send"); return nil }), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err = p.Run(ctx); !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if s.Status().InFlight != 0 {
			t.Fatal(s.Status())
		}
	}
}
