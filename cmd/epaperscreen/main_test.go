package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenusbhost"
)

type fakeTransport struct {
	ready   func() (screendelivery.Readiness, error)
	send    func(display.Frame) error
	inspect func() (screenusbhost.Snapshot, error)
	closed  bool
}

func (*fakeTransport) Changed() <-chan struct{} { return nil }
func (*fakeTransport) ResetSession()            { panic("CLI must not reset or replay") }
func (s *fakeTransport) Close() error           { s.closed = true; return nil }
func (s *fakeTransport) WaitReady(context.Context) (screendelivery.Readiness, error) {
	return s.ready()
}
func (s *fakeTransport) Send(_ context.Context, f display.Frame) error           { return s.send(f) }
func (s *fakeTransport) Inspect(context.Context) (screenusbhost.Snapshot, error) { return s.inspect() }

func cliFixture(t *testing.T, s *fakeTransport) (func(), string) {
	t.Helper()
	old := openTransport
	openTransport = func(string, display.Size) (transport, error) { return s, nil }
	path := t.TempDir() + "/scene.html"
	if err := os.WriteFile(path, []byte("<p>Original</p>"), 0600); err != nil {
		t.Fatal(err)
	}
	return func() { openTransport = old }, path
}

func TestCLIWaitsForReadinessBeforeReadingAndRenderingLatest(t *testing.T) {
	s := &fakeTransport{}
	restore, path := cliFixture(t, s)
	defer restore()
	oldRender := render
	defer func() { render = oldRender }()
	ready, sends := false, 0
	s.ready = func() (screendelivery.Readiness, error) {
		ready = true
		if err := os.WriteFile(path, []byte("<p>Latest</p>"), 0600); err != nil {
			t.Fatal(err)
		}
		return screendelivery.Readiness{}, nil
	}
	render = func(_ context.Context, c renderConfig, source []byte) (rendered, error) {
		if !ready || string(source) != "<p>Latest</p>" {
			t.Fatal("stale/pre-ready render")
		}
		frame, err := display.NewFrame(c.size, 1, []byte{0x80})
		return rendered{frame: frame}, err
	}
	s.send = func(display.Frame) error { sends++; return nil }
	var output bytes.Buffer
	if err := run(context.Background(), []string{"-width", "8", "-height", "1", path, "port"}, &output); err != nil {
		t.Fatal(err)
	}
	if sends != 1 || !s.closed || !bytes.Contains(output.Bytes(), []byte("confirmed")) {
		t.Fatal(sends, s.closed, output.String())
	}
}

func TestCLILostACKOnlyReconcilesNeverResends(t *testing.T) {
	for _, confirmed := range []bool{true, false} {
		t.Run(map[bool]string{true: "confirmed", false: "unknown"}[confirmed], func(t *testing.T) {
			s := &fakeTransport{}
			restore, path := cliFixture(t, s)
			defer restore()
			calls, sends := 0, 0
			s.ready = func() (screendelivery.Readiness, error) {
				calls++
				if calls == 1 {
					return firstReadiness(), nil
				}
				if sends == 0 {
					return screendelivery.Readiness{}, nil
				}
				outcome := screendelivery.PendingUnconfirmed
				if confirmed {
					outcome = screendelivery.PendingConfirmed
				}
				return screendelivery.Readiness{Pending: outcome}, nil
			}
			s.send = func(display.Frame) error { sends++; return screendelivery.ErrTransportLost }
			var output bytes.Buffer
			err := run(context.Background(), []string{path, "port"}, &output)
			assertCLIOutcome(t, confirmed, err, sends, s.closed)
		})
	}
}

func assertCLIOutcome(t *testing.T, confirmed bool, err error, sends int, closed bool) {
	t.Helper()
	if confirmed && err != nil {
		t.Fatal(err)
	}
	if !confirmed && !errors.Is(err, ErrUnconfirmed) {
		t.Fatal(err)
	}
	if sends != 1 || !closed {
		t.Fatal(sends, closed)
	}
}

func firstReadiness() screendelivery.Readiness {
	return screendelivery.Readiness{NotBefore: time.Now().Add(time.Millisecond)}
}
