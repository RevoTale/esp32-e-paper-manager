package streamsession

import (
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestCooldownSurvivesOwnershipAndRejectsBeforeSPI(t *testing.T) {
	s, sink, tx := fixture(t)
	if err := s.Begin(s.writer, tx, time.Second-1); !errors.Is(err, ErrCooldown) || sink.begins != 0 {
		t.Fatal(sink, err)
	}
	ready(t, s, tx, time.Second)
	if err := s.Commit(s.writer, tx, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := s.Disconnect(s.writer); err != nil {
		t.Fatal(err)
	}
	lease, err := s.Acquire(s.Epoch(), [16]byte{2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Bind(lease); err != nil {
		t.Fatal(err)
	}
	tx.Lease = lease
	if err = s.Begin(s.writer, tx, 2*time.Second-1); !errors.Is(err, ErrCooldown) {
		t.Fatal(err)
	}
	ready(t, s, tx, 2*time.Second)
}

func TestFailedBeginConsumesIDAndRetainsCause(t *testing.T) {
	s, sink, tx := fixture(t)
	sink.errorBegin = errors.New("initialization failed")
	if err := s.Begin(s.writer, tx, time.Second); !errors.Is(err, sink.errorBegin) {
		t.Fatal(err)
	}
	if err := s.Begin(s.writer, tx, time.Second); !errors.Is(err, ErrStale) {
		t.Fatal(err)
	}
	result, err := s.Query(s.writer, tx)
	if err != nil || result.Status.State != streamrx.Failed || !errors.Is(result.Status.Failure, sink.errorBegin) || sink.aborts != 1 {
		t.Fatal(result, sink, err)
	}
}

func TestIdleDeadlineAndClockRegression(t *testing.T) {
	for _, now := range []time.Duration{-1, time.Second - 1, 2 * time.Second} {
		s, sink, tx := fixture(t)
		if err := s.Begin(s.writer, tx, time.Second); err != nil {
			t.Fatal(err)
		}
		if err := s.Tick(now); !errors.Is(err, streamrx.ErrTimeout) || sink.aborts != 1 {
			t.Fatal(now, sink, err)
		}
		result, err := s.Query(s.writer, tx)
		if err != nil || result.Status.State != streamrx.Failed {
			t.Fatal(result, err)
		}
		if err = s.Disconnect(s.writer); err != nil {
			t.Fatal(err)
		}
		if err = s.Tick(0); !errors.Is(err, streamrx.ErrTimeout) {
			t.Fatal("clock reset", err)
		}
	}
}

func TestCommitRecheckDeadlineAndConflictingWrites(t *testing.T) {
	s, sink, tx := fixture(t)
	ready(t, s, tx, time.Second)
	if err := s.Commit(s.writer, tx, 2*time.Second); !errors.Is(err, streamrx.ErrTimeout) || sink.commits != 0 {
		t.Fatal(sink, err)
	}
	tx.ID++
	if err := s.Begin(s.writer, tx, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	bad := tx
	bad.Digest[0] ^= 1
	if _, err := s.Query(s.writer, bad); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if err := s.Write(s.writer, bad, 0, 0, []byte{128}, 2*time.Second); !errors.Is(err, streamrx.ErrState) {
		t.Fatal(err)
	}
	if sink.commits != 0 || sink.aborts != 2 {
		t.Fatal(sink)
	}
}

func TestUnknownTransactionDoesNotTouchSPI(t *testing.T) {
	s, sink, a := fixture(t)
	result, err := s.Query(s.writer, a)
	if err != nil || result.Status.State != streamrx.Idle {
		t.Fatal(result, err)
	}
	if err = s.Commit(s.writer, a, time.Second); !errors.Is(err, streamrx.ErrState) {
		t.Fatal(err)
	}
	if err = s.Write(s.writer, a, 0, 0, []byte{128}, time.Second); !errors.Is(err, ErrStale) {
		t.Fatal(err)
	}
	if err = s.Abort(s.writer); err != nil {
		t.Fatal(err)
	}
	if sink.begins+sink.writes+sink.commits+sink.aborts != 0 {
		t.Fatal(sink)
	}
}

func TestCompletedDuringNewUpload(t *testing.T) {
	s, sink, a := fixture(t)
	ready(t, s, a, time.Second)
	if err := s.Commit(s.writer, a, time.Second); err != nil {
		t.Fatal(err)
	}
	b := a
	b.ID++
	if err := s.Begin(s.writer, b, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := s.Begin(s.writer, b, 2*time.Second); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	if err := s.Commit(s.writer, a, 2*time.Second); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	if err := s.Abort(s.writer); err != nil {
		t.Fatal(err)
	}
	result, err := s.Query(s.writer, b)
	if err != nil || result.Status.State != streamrx.Closed || sink.commits != 1 || sink.aborts != 1 {
		t.Fatal(result, sink, err)
	}
}

func TestExpiredInputCannotStageNewBytes(t *testing.T) {
	s, sink, tx := fixture(t)
	if err := s.Begin(s.writer, tx, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := s.Write(s.writer, tx, 0, 0, []byte{128}, 2*time.Second); !errors.Is(err, streamrx.ErrTimeout) || sink.writes != 0 {
		t.Fatal(sink, err)
	}
	if err := s.Begin(s.writer, tx, time.Second); !errors.Is(err, streamrx.ErrTimeout) {
		t.Fatal(err)
	}
}
