package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"io"
	"strings"
	"testing"
)

type panelTransport struct {
	fakeTransport
	report screenwire.PanelStatus
	err    error
}

func (p *panelTransport) InspectPanel(context.Context) (screenwire.PanelStatus, error) {
	return p.report, p.err
}

func TestPanelCommandReportsEvidenceNotPixels(t *testing.T) {
	p := &panelTransport{report: screenwire.PanelStatus{Version: 1, State: 2, Cycle: 2,
		Waits: [3]screenwire.BusySamples{{Samples: 1}, {Samples: 3, LowSamples: 2}, {Samples: 1}}}}
	old := openTransport
	openTransport = func(string, display.Size) (transport, error) { return p, nil }
	t.Cleanup(func() { openTransport = old })
	var output bytes.Buffer
	if err := run(context.Background(), []string{"--panel-status", "port"}, &output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"cycle=2", "refresh_samples=3 refresh_low=2", "pixels=unverified"} {
		if !strings.Contains(output.String(), want) {
			t.Fatal(output.String())
		}
	}
	if !p.closed {
		t.Fatal("not closed")
	}
	if err := runPanelStatus(context.Background(), []string{"port"}, statusBrokenWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	p.err = io.EOF
	if err := runPanelStatus(context.Background(), []string{"port"}, &output); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
}

func TestPanelCommandRejectsArgumentsAndUnsupportedTransport(t *testing.T) {
	if err := runPanelStatus(context.Background(), nil, io.Discard); err == nil {
		t.Fatal("missing port")
	}
	restore, _ := cliFixture(t, &fakeTransport{})
	defer restore()
	if err := runPanelStatus(context.Background(), []string{"port"}, io.Discard); err == nil {
		t.Fatal("unsupported")
	}
	openTransport = func(string, display.Size) (transport, error) { return nil, io.EOF }
	if err := runPanelStatus(context.Background(), []string{"port"}, io.Discard); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
}
