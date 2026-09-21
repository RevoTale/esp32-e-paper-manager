package screenclient

import (
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func rewriteStatus(t *testing.T, rw *transport, kind screenwire.Kind, mutate func(*screenwire.Status)) {
	t.Helper()
	rw.rewrite = func(request screenwire.Record, p []byte) []byte {
		if request.Kind != kind {
			return p
		}
		r, err := screenwire.Decode(p)
		if err != nil {
			t.Fatal(err)
		}
		status, err := screenwire.DecodeStatus(r.Payload[:screenwire.StatusSize])
		if err != nil {
			t.Fatal(err)
		}
		mutate(&status)
		body := append([]byte(nil), r.Payload...)
		if err = screenwire.EncodeStatus(body[:screenwire.StatusSize], status); err != nil {
			t.Fatal(err)
		}
		r.Payload = body
		out := make([]byte, len(p))
		if _, err = screenwire.Encode(out, r); err != nil {
			t.Fatal(err)
		}
		return out
	}
}

func TestInvalidAcknowledgementsNeverConfirm(t *testing.T) {
	cases := []struct {
		kind   screenwire.Kind
		mutate func(*screenwire.Status)
	}{
		{screenwire.Begin, func(s *screenwire.Status) { s.State = streamrx.Idle }},
		{screenwire.Data, func(s *screenwire.Status) { s.Offset++ }},
		{screenwire.Commit, func(s *screenwire.Status) { s.CurrentImage = false }},
		{screenwire.Commit, func(s *screenwire.Status) { s.Pass = 1 }},
	}
	for _, test := range cases {
		c, _, rw, _, f := fixture(t)
		rewriteStatus(t, rw, test.kind, test.mutate)
		if err := c.Send(f); !errors.Is(err, screenwire.ErrRecord) || !c.Pending() {
			t.Fatal(test.kind, err)
		}
	}
}

func TestFrameChecksBeforeAnyBegin(t *testing.T) {
	c, _, _, sink, frame := fixture(t)
	if err := c.Send(display.Frame{}); !errors.Is(err, screenwire.ErrRecord) {
		t.Fatal(err)
	}
	frame.Bytes()[2] = 1
	if err := c.Send(frame); !errors.Is(err, screenwire.ErrRecord) {
		t.Fatal(err)
	}
	frame.Bytes()[2] = 128
	c.next = ^uint64(0)
	if err := c.Send(frame); !errors.Is(err, screenwire.ErrRecord) {
		t.Fatal(err)
	}
	if sink.commits != 0 || c.Pending() {
		t.Fatal(sink)
	}
}

func TestCapsBoundReplyProgress(t *testing.T) {
	c, _, _, _, _ := fixture(t)
	for _, s := range []screenwire.Status{
		{State: streamrx.Complete, Pass: 1}, {State: streamrx.Receiving, Pass: 2},
		{State: streamrx.Failed, Pass: 3}, {State: streamrx.Receiving, Offset: 27},
	} {
		if c.validProgress(s) {
			t.Fatal(s)
		}
	}
	if !c.validProgress(screenwire.Status{State: streamrx.Failed, Pass: 2}) {
		t.Fatal("valid late failure rejected")
	}
}

func TestReconcileRejectsChangedIdentityOrConflict(t *testing.T) {
	for _, mutate := range []func(*screenwire.Status){
		func(s *screenwire.Status) { s.Boot[0]++ },
		func(s *screenwire.Status) { s.Generation++ },
		func(s *screenwire.Status) { s.Code = screenwire.CodeConflict },
	} {
		c, _, rw, _, frame := fixture(t)
		if err := c.start(frame); err != nil {
			t.Fatal(err)
		}
		rewriteStatus(t, rw, screenwire.Query, mutate)
		if _, err := c.Reconcile(); err == nil || !c.Pending() {
			t.Fatal(err)
		}
	}
}

func TestReconciliationGuardsAndAbortFailure(t *testing.T) {
	c, _, rw, _, frame := fixture(t)
	if _, err := c.Reconcile(); !errors.Is(err, ErrPending) {
		t.Fatal(err)
	}
	if err := c.start(frame); err != nil {
		t.Fatal(err)
	}
	rw.dropKind = screenwire.Abort
	if _, err := c.Reconcile(); err == nil || !c.Pending() {
		t.Fatal(err)
	}
	if _, err := c.Reconcile(); !errors.Is(err, ErrConnection) {
		t.Fatal(err)
	}
	if _, err := c.exchange(screenwire.Record{Kind: screenwire.Hello}); !errors.Is(err, ErrConnection) {
		t.Fatal(err)
	}
}

func TestRemoteErrorIsSourceFree(t *testing.T) {
	err := RemoteError{Status: screenwire.Status{Operation: screenwire.Commit, Code: screenwire.CodeHardware,
		Diagnostic: screenwire.Diagnostic{Phase: 5, Step: 14, Command: 0x12, BusyKnown: true}}}
	want := "screen rejected: operation=6 code=11 state=0 pass=0 offset=0 phase=5 step=14 command=12 busy_known=true busy=false"
	if err.Error() != want {
		t.Fatal(err)
	}
	for _, code := range []screenwire.Code{screenwire.CodeStale, screenwire.CodeHardware, screenwire.CodeTimeout, screenwire.CodeChunk, screenwire.CodeDigest, screenwire.CodeState} {
		if !reconcilable(code) {
			t.Fatal(code)
		}
	}
}
