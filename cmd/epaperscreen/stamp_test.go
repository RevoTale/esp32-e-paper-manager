package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func TestUSBRendersReservedTimestampInExplicitZone(t *testing.T) {
	s := &fakeTransport{ready: func() (screendelivery.Readiness, error) { return screendelivery.Readiness{}, nil }}
	restore, path := cliFixture(t, s)
	defer restore()
	sent := false
	s.send = func(frame display.Frame) error {
		sent = true
		area, err := refreshstamp.Bounds(frame.Size())
		if err != nil || frame.Pixel(area.Min.X, area.Min.Y) != display.Black {
			t.Fatal("missing reserved full-cycle timestamp", err)
		}
		return nil
	}
	if err := run(context.Background(), []string{"-timezone", "UTC", path, "port"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !sent || !s.closed {
		t.Fatal("expected one delivery and transport cleanup")
	}
}

func TestTimestampUsesReadyClockAndSelectedZone(t *testing.T) {
	for _, zone := range []string{"Europe/Kiev", "UTC", "America/New_York"} {
		t.Run(zone, func(t *testing.T) {
			oldClock := wallNow
			defer func() { wallNow = oldClock }()
			ready := false
			instant := time.Date(2026, 9, 7, 22, 12, 0, 0, time.UTC)
			wallNow = func() time.Time {
				if !ready {
					t.Fatal("clock captured before readiness")
				}
				return instant
			}
			s := &fakeTransport{ready: func() (screendelivery.Readiness, error) {
				ready = true
				return screendelivery.Readiness{}, nil
			}}
			restore, path := cliFixture(t, s)
			defer restore()
			s.send = func(frame display.Frame) error {
				assertTimestamp(t, frame, zone, instant)
				return nil
			}
			args := []string{path, "port"}
			if zone != "Europe/Kiev" {
				args = append([]string{"-timezone", zone}, args...)
			}
			if err := run(context.Background(), args, io.Discard); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func assertTimestamp(t *testing.T, frame display.Frame, zone string, instant time.Time) {
	t.Helper()
	location, err := time.LoadLocation(zone)
	if err != nil {
		t.Fatal(err)
	}
	tracker, err := refreshstamp.New(location)
	if err != nil {
		t.Fatal(err)
	}
	label, err := tracker.Begin(1, instant)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := display.NewFrame(frame.Size(), frame.Stride(), make([]byte, len(frame.Bytes())))
	if err != nil {
		t.Fatal(err)
	}
	if err := refreshstamp.Paint(expected, label); err != nil {
		t.Fatal(err)
	}
	area, err := refreshstamp.Bounds(frame.Size())
	if err != nil {
		t.Fatal(err)
	}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			if frame.Pixel(x, y) != expected.Pixel(x, y) {
				t.Fatalf("timestamp pixel %d,%d", x, y)
			}
		}
	}
}

func TestInvalidZonesFailBeforeOpeningUSB(t *testing.T) {
	old := openTransport
	defer func() { openTransport = old }()
	openTransport = func(string, display.Size) (transport, error) { t.Fatal("USB opened"); return nil, nil }
	for _, zone := range []string{"", "not/a/timezone"} {
		if err := run(context.Background(), []string{"-timezone", zone, "file", "port"}, io.Discard); !errors.Is(err, refreshstamp.ErrConfiguration) {
			t.Fatal(err)
		}
	}
}

func TestTimestampValidationFailsWithoutDelivering(t *testing.T) {
	c := renderConfig{size: display.Size{Width: 800, Height: 480}}
	if _, err := renderHTML(context.Background(), c, nil); !errors.Is(err, refreshstamp.ErrConfiguration) {
		t.Fatal(err)
	}
	c.zone = time.UTC
	old := wallNow
	defer func() { wallNow = old }()
	wallNow = func() time.Time { return time.Time{} }
	if _, err := renderHTML(context.Background(), c, nil); !errors.Is(err, refreshstamp.ErrTime) {
		t.Fatal(err)
	}
	wallNow = time.Now
	c.size = display.Size{Width: 8, Height: 1}
	if _, err := renderHTML(context.Background(), c, nil); !errors.Is(err, display.ErrFrameGeometry) {
		t.Fatal(err)
	}
}

func TestReservedOverlapWarningAndDiagnosticsFailure(t *testing.T) {
	s := &fakeTransport{ready: func() (screendelivery.Readiness, error) { return screendelivery.Readiness{}, nil }}
	restore, path := cliFixture(t, s)
	defer restore()
	old := render
	defer func() { render = old }()
	render = func(ctx context.Context, c renderConfig, _ []byte) (rendered, error) {
		return renderHTML(ctx, c, []byte(`<div style="position:absolute;right:0;bottom:0;width:100%;height:40px;background:black"></div>`))
	}
	sends := 0
	s.send = func(display.Frame) error { sends++; return nil }
	var out bytes.Buffer
	if err := run(context.Background(), []string{path, "port"}, &out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("warning=reserved-overlap")) || sends != 1 {
		t.Fatal(out.String(), sends)
	}
	if err := run(context.Background(), []string{path, "port"}, diagnosticFailure{}); !errors.Is(err, io.ErrClosedPipe) || sends != 1 {
		t.Fatal(err, sends)
	}
}

type diagnosticFailure struct{}

func (diagnosticFailure) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
