package manager

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func rejectionFixture(t *testing.T) (*Screen, *renderdiag.Error) {
	t.Helper()
	diagnostic := &renderdiag.Error{Code: renderdiag.InvalidValue, Source: renderdiag.Inline, Element: 4, Start: 6, End: 12}
	s := screenFixture(t, screenRenderer(func(ctx context.Context, size display.Size, html []byte) (display.Frame, error) {
		if string(html) == "private-invalid" {
			return display.Frame{}, diagnostic
		}
		return blackRenderer(ctx, size, html)
	}))
	return s, diagnostic
}

func rejectedScreen(t *testing.T) (*Screen, *renderdiag.Error) {
	t.Helper()
	s, diagnostic := rejectionFixture(t)
	if _, err := s.Submit(0, []byte("good"), 0); err != nil {
		t.Fatal(err)
	}
	d, err := s.RenderNext(context.Background(), 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Resolve(d.Revision, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Submit(1, []byte("private-invalid"), 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RenderNext(context.Background(), 1, true); !errors.Is(err, renderdiag.ErrRejected) {
		t.Fatal(err)
	}
	return s, diagnostic
}

func TestScreenRejectionPreservesConfirmedAndCopiesDiagnostic(t *testing.T) {
	s, diagnostic := rejectedScreen(t)
	status := s.Status()
	if status.Confirmed != 1 || status.Current != 2 || status.InFlight != 0 || status.Failure == nil || status.Failure.Revision != 2 {
		t.Fatal(status)
	}
	diagnostic.Element = 99
	status.Failure.Diagnostic.Element = 98
	if s.Status().Failure.Diagnostic.Element != 4 {
		t.Fatal("mutable diagnostic alias")
	}
}

func TestScreenRejectionWaitsForExplicitCorrection(t *testing.T) {
	s, _ := rejectedScreen(t)
	if d, err := s.RenderNext(context.Background(), 2, true); err != nil || d.Revision != 0 {
		t.Fatal("retried invalid document", d, err)
	}
	if _, err := s.Submit(2, []byte("fixed"), 2); err != nil {
		t.Fatal(err)
	}
	if s.Status().Failure != nil {
		t.Fatal("stale diagnostic on new scene")
	}
}

func TestScreenAPIRejectionIsSourceFreeAndRevisionBound(t *testing.T) {
	s, _ := rejectionFixture(t)
	a, err := NewScreenAPI(s, []byte(strings.Repeat("t", 32)), [16]byte{1}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if w := apiRequest(a, "PUT", "/v2/screen", a.tag(0), "private-invalid"); w.Code != 202 {
		t.Fatal(w)
	}
	if _, err := s.RenderNext(context.Background(), time.Second, true); !errors.Is(err, renderdiag.ErrRejected) {
		t.Fatal(err)
	}
	w := apiRequest(a, "GET", "/v2/screen/status", "", "")
	var state ScreenStatus
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.Failure == nil || state.Failure.Revision != 1 || state.Failure.Diagnostic.Code != renderdiag.InvalidValue {
		t.Fatal(w.Body.String())
	}
	if strings.Contains(w.Body.String(), "private-invalid") {
		t.Fatal("source leak")
	}
}

func TestScreenPumpWaitsForNewSceneAfterRejection(t *testing.T) {
	s, _ := rejectionFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sent := 0
	p, err := NewScreenPump(s, screenSender(func(context.Context, display.Frame) error { sent++; cancel(); return nil }), time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Submit(0, []byte("private-invalid"), 0); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()
	waitRejection(t, ctx, s, done)
	if _, err := s.submitAt(1, []byte("fixed"), s.elapsed); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if sent != 1 || s.Status().Confirmed != 2 || s.Status().Failure != nil {
		t.Fatal(sent, s.Status())
	}
}

func waitRejection(t *testing.T, ctx context.Context, s *Screen, done <-chan error) {
	t.Helper()
	for s.Status().Failure == nil {
		select {
		case err := <-done:
			t.Fatal("pump stopped on document rejection", err)
		case <-ctx.Done():
			t.Fatal("missing rejection diagnostic")
		case <-time.After(time.Millisecond):
		}
	}
}

func TestSupersededRejectionCannotStainNewRevision(t *testing.T) {
	var s *Screen
	s = screenFixture(t, screenRenderer(func(context.Context, display.Size, []byte) (display.Frame, error) {
		if _, err := s.Submit(1, []byte("fixed"), 1); err != nil {
			t.Fatal(err)
		}
		return display.Frame{}, &renderdiag.Error{Code: renderdiag.Syntax}
	}))
	if _, err := s.Submit(0, []byte("bad"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RenderNext(context.Background(), 0, true); !errors.Is(err, ErrSuperseded) {
		t.Fatal(err)
	}
	if s.Status().Failure != nil || s.Status().Current != 2 {
		t.Fatal(s.Status())
	}
}

func TestQueueFailureCannotMasqueradeAsDocumentRejection(t *testing.T) {
	var s *Screen
	s = screenFixture(t, screenRenderer(func(context.Context, display.Size, []byte) (display.Frame, error) {
		// Deliberately violate ownership to test the invariant-failure boundary.
		if err := s.queue.Release(1); err != nil {
			t.Fatal(err)
		}
		return display.Frame{}, &renderdiag.Error{Code: renderdiag.Syntax}
	}))
	if _, err := s.Submit(0, []byte("bad"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RenderNext(context.Background(), 0, true); err == nil || errors.Is(err, renderdiag.ErrRejected) {
		t.Fatal(err)
	}
	if s.Status().Failure != nil {
		t.Fatal("recorded handled rejection after queue failure")
	}
}

func TestNewSceneCannotHideFatalRendererFailure(t *testing.T) {
	var s *Screen
	crash := errors.New("renderer process failed")
	calls := 0
	s = screenFixture(t, screenRenderer(func(context.Context, display.Size, []byte) (display.Frame, error) {
		calls++
		if calls == 1 {
			if _, err := s.submitAt(1, []byte("new"), s.elapsed); err != nil {
				t.Fatal(err)
			}
		}
		return display.Frame{}, crash
	}))
	if _, err := s.Submit(0, []byte("old"), 0); err != nil {
		t.Fatal(err)
	}
	p, err := NewScreenPump(s, screenSender(func(context.Context, display.Frame) error { t.Error("unexpected send"); return nil }), time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Run(context.Background()); !errors.Is(err, crash) || errors.Is(err, ErrSuperseded) {
		t.Fatal(err)
	}
	if calls != 1 || s.Status().Current != 2 || s.Status().Failure != nil {
		t.Fatal(calls, s.Status())
	}
}

func TestNewSceneCannotHideInvalidRendererFrame(t *testing.T) {
	var s *Screen
	s = screenFixture(t, screenRenderer(func(context.Context, display.Size, []byte) (display.Frame, error) {
		if _, err := s.Submit(1, []byte("new"), 1); err != nil {
			t.Fatal(err)
		}
		return display.Frame{}, nil
	}))
	if _, err := s.Submit(0, []byte("old"), 0); err != nil {
		t.Fatal(err)
	}
	_, err := s.RenderNext(context.Background(), 0, true)
	if !errors.Is(err, display.ErrFrameGeometry) || errors.Is(err, ErrSuperseded) {
		t.Fatal(err)
	}
}
