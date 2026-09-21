package screenclient

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestLostAcquireACKRetriesOriginalClaim(t *testing.T) {
	_, d, old, _, _ := fixture(t)
	if err := old.connection.Disconnect(); err != nil {
		t.Fatal(err)
	}
	c, err := New(bytes.NewReader(bytes.Repeat([]byte{3}, 16)))
	if err != nil {
		t.Fatal(err)
	}
	rw := &transport{connection: d.Open(), now: time.Second, dropKind: screenwire.Acquire}
	if _, err = c.Connect(rw); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal(err)
	}
	claim := c.acquiring
	if claim.epoch.Generation != 1 || c.lease.Generation != 0 {
		t.Fatal("grant adopted without ACK")
	}
	rw = &transport{connection: d.Open(), now: time.Second}
	if _, err = c.Connect(rw); err != nil {
		t.Fatal(err)
	}
	if c.lease.Generation != 2 || c.lease.Claim != claim.claim {
		t.Fatal("retry changed owner")
	}
}

func TestLostAcquireDoesNotAdoptCompetingWinner(t *testing.T) {
	_, d, old, _, _ := fixture(t)
	if err := old.connection.Disconnect(); err != nil {
		t.Fatal(err)
	}
	c, err := New(bytes.NewReader(bytes.Repeat([]byte{3}, 16)))
	if err != nil {
		t.Fatal(err)
	}
	rw := &transport{connection: d.Open(), now: time.Second, dropKind: screenwire.Acquire, before: true}
	if _, err = c.Connect(rw); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal(err)
	}
	winner := randomClient(t, 4)
	rw = &transport{connection: d.Open(), now: time.Second}
	if _, err = winner.Connect(rw); err != nil {
		t.Fatal(err)
	}
	if err = rw.connection.Disconnect(); err != nil {
		t.Fatal(err)
	}
	_, err = c.Connect(&transport{connection: d.Open(), now: time.Second})
	var remote RemoteError
	if !errors.As(err, &remote) || remote.Status.Code != screenwire.CodeLease || c.lease.Generation != 0 {
		t.Fatal(err)
	}
	if !errors.Is(err, ErrResync) {
		t.Fatal("competing ownership treated as transient", err)
	}
}

func randomClient(t *testing.T, value byte) *Client {
	t.Helper()
	c, err := New(bytes.NewReader(bytes.Repeat([]byte{value}, 16)))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestInterruptedUpdateReconcilesBeforeNewID(t *testing.T) {
	for _, kind := range []screenwire.Kind{screenwire.Begin, screenwire.Data} {
		c, d, rw, sink, frame := fixture(t)
		rw.dropKind = kind
		if err := c.Send(frame); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatal(err)
		}
		if err := c.Send(frame); !errors.Is(err, ErrConnection) {
			t.Fatal(err)
		}
		rw = &transport{connection: d.Open(), now: time.Second}
		if _, err := c.Connect(rw); err != nil {
			t.Fatal(err)
		}
		if err := c.Send(frame); !errors.Is(err, ErrPending) {
			t.Fatal(err)
		}
		result, err := c.Reconcile()
		wantUnconfirmed(t, c, result, err)
		if err = c.Send(frame); err != nil {
			t.Fatal(err)
		}
		if c.next != 2 || sink.commits != 1 {
			t.Fatal(c.next, sink)
		}
	}
}

func wantUnconfirmed(t *testing.T, c *Client, result Reconciliation, err error) {
	t.Helper()
	if err != nil || result.Confirmed || c.Pending() {
		t.Fatal(result, err)
	}
}

func TestPendingActiveUploadIsAbortedBeforeRelease(t *testing.T) {
	c, _, _, sink, frame := fixture(t)
	if err := c.start(frame); err != nil {
		t.Fatal(err)
	}
	result, err := c.Reconcile()
	if err != nil || result.Confirmed || c.Pending() || sink.commits != 0 {
		t.Fatal(result, err)
	}
	if err = c.Send(frame); err != nil {
		t.Fatal(err)
	}
}

func TestCooldownRejectionCanBeReconciledWithoutRefresh(t *testing.T) {
	c, _, _, sink, frame := fixture(t)
	if err := c.Send(frame); err != nil {
		t.Fatal(err)
	}
	var remote RemoteError
	if err := c.Send(frame); !errors.As(err, &remote) || remote.Status.Code != screenwire.CodeCooldown {
		t.Fatal(err)
	}
	result, err := c.Reconcile()
	if err != nil || result.Confirmed || sink.commits != 1 {
		t.Fatal(result, err)
	}
}
