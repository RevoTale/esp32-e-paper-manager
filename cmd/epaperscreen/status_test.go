package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenusbhost"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestStatusDoesNotMisrepresentReplyEnvelopeAsTransferEvidence(t *testing.T) {
	s := &fakeTransport{inspect: func() (screenusbhost.Snapshot, error) {
		return screenusbhost.Snapshot{Status: screenwire.Status{
			State: streamrx.Failed, Pass: 1, Offset: 1000,
			Boot: [16]byte{'p', 'r', 'i', 'v', 'a', 't', 'e'}, Generation: 12345,
			Diagnostic: screenwire.Diagnostic{Domain: screenwire.DomainPanel,
				Code: 3, Phase: 2, Step: 4, Command: 0x12, Offset: -1, BusyKnown: true},
		}}, nil
	}}
	restore, _ := cliFixture(t, s)
	defer restore()
	var output bytes.Buffer
	if err := runStatus(context.Background(), []string{"port"}, &output); err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{"pass=1 offset=1000", "current_image=", "domain=1 code=3", "busy_known="} {
		if strings.Contains(output.String(), absent) {
			t.Fatalf("non-transaction inspection misrepresented %q", output.String())
		}
	}
	if !strings.Contains(output.String(), "transfer_evidence=unavailable") {
		t.Fatal("missing explicit evidence boundary", output.String())
	}
	if strings.Contains(output.String(), "private") || strings.Contains(output.String(), "12345") || !s.closed {
		t.Fatal("identity emitted or transport not closed", output.String())
	}
}

func TestStatusPropagatesDiagnosticOutputFailure(t *testing.T) {
	s := &fakeTransport{inspect: func() (screenusbhost.Snapshot, error) { return screenusbhost.Snapshot{}, nil }}
	restore, _ := cliFixture(t, s)
	defer restore()
	if err := runStatus(context.Background(), []string{"port"}, statusBrokenWriter{}); !errors.Is(err, io.ErrClosedPipe) || !s.closed {
		t.Fatal(err, s.closed)
	}
}

type statusBrokenWriter struct{}

func (statusBrokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
