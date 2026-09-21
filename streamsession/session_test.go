package streamsession

import (
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

type testSink struct {
	begins, writes, commits, aborts int
	commitErr, errorBegin           error
}

func (s *testSink) Begin() error                      { s.begins++; return s.errorBegin }
func (s *testSink) Write(uint8, uint32, []byte) error { s.writes++; return nil }
func (s *testSink) Commit() error                     { s.commits++; return s.commitErr }
func (s *testSink) Abort() error                      { s.aborts++; return nil }

func fixture(t *testing.T) (*Session, *testSink, Transaction) {
	t.Helper()
	sink := &testSink{}
	s, err := New([16]byte{1}, streamrx.Config{Width: 8, Height: 1, Passes: 2, MaxChunk: 1, Idle: time.Second, Total: 2 * time.Second}, sink, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := s.Acquire(s.Epoch(), [16]byte{1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Bind(lease); err != nil {
		t.Fatal(err)
	}
	return s, sink, Transaction{Lease: lease, ID: 1, Digest: sha256.Sum256([]byte{128})}
}

func ready(t *testing.T, s *Session, tx Transaction, now time.Duration) {
	t.Helper()
	if err := s.Begin(s.writer, tx, now); err != nil {
		t.Fatal(err)
	}
	for pass := uint8(0); pass < 2; pass++ {
		if err := s.Write(s.writer, tx, pass, 0, []byte{128}, now); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLostACKRetainsCompletionWithoutRepeatSPI(t *testing.T) {
	s, sink, tx := fixture(t)
	ready(t, s, tx, time.Second)
	if err := s.Commit(s.writer, tx, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := s.Disconnect(s.writer); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Bind(tx.Lease); err != nil {
		t.Fatal(err)
	}
	result, err := s.Query(s.writer, tx)
	if err != nil || result.Status.State != streamrx.Complete || !result.CurrentImage {
		t.Fatal(result, err)
	}
	if err = s.Commit(s.writer, tx, 2*time.Second); err != nil || sink.commits != 1 {
		t.Fatal(sink, err)
	}
	bad := tx
	bad.Digest[0] ^= 1
	if _, err = s.Query(s.writer, bad); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}

func TestAbortedIDsCannotRestart(t *testing.T) {
	s, sink, tx := fixture(t)
	if err := s.Begin(s.writer, tx, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := s.Disconnect(s.writer); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Bind(tx.Lease); err != nil {
		t.Fatal(err)
	}
	if err := s.Begin(s.writer, tx, time.Second); !errors.Is(err, ErrStale) {
		t.Fatal(err)
	}
	result, err := s.Query(s.writer, tx)
	if err != nil || result.Status.State != streamrx.Closed || result.CurrentImage {
		t.Fatal(result, err)
	}
	tx.ID++
	ready(t, s, tx, 2*time.Second)
	if sink.begins != 2 || sink.aborts != 1 {
		t.Fatal(sink)
	}
}

func TestOwnershipHandoverFencesOldGeneration(t *testing.T) {
	s, _, a := fixture(t)
	if err := s.Disconnect(s.writer); err != nil {
		t.Fatal(err)
	}
	leaseB, err := s.Acquire(a.Epoch, [16]byte{2})
	if err != nil || leaseB.Generation != 2 {
		t.Fatal(leaseB, err)
	}
	if _, err = s.Acquire(a.Epoch, [16]byte{3}); !errors.Is(err, ErrLease) {
		t.Fatal(err)
	}
	if _, err = s.Bind(leaseB); err != nil {
		t.Fatal(err)
	}
	if err = s.Begin(s.writer, a, time.Second); !errors.Is(err, ErrLease) {
		t.Fatal(err)
	}
	b := a
	b.Lease = leaseB
	if err = s.Begin(s.writer, b, time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Acquire(leaseB.Epoch, [16]byte{3}); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
}

func TestLateCommitFailureInvalidatesImageProof(t *testing.T) {
	s, sink, a := fixture(t)
	ready(t, s, a, time.Second)
	if err := s.Commit(s.writer, a, time.Second); err != nil {
		t.Fatal(err)
	}
	b := a
	b.ID = 2
	ready(t, s, b, 2*time.Second)
	sink.commitErr = errors.New("late power-off failure")
	if err := s.Commit(s.writer, b, 2*time.Second); err == nil {
		t.Fatal("failure lost")
	}
	result, err := s.Query(s.writer, b)
	if err != nil || result.Status.State != streamrx.Failed || result.CurrentImage {
		t.Fatal(result, err)
	}
	if _, err = s.Query(s.writer, a); !errors.Is(err, ErrStale) {
		t.Fatal("evicted old evidence", err)
	}
	if err = s.Commit(s.writer, b, 3*time.Second); err == nil || sink.commits != 2 {
		t.Fatal(sink, err)
	}
}
