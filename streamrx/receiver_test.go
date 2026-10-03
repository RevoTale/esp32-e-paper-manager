package streamrx

import (
	"crypto/sha256"
	"errors"
	"testing"
	"time"
)

type sink struct {
	begin, writes, commits, aborts          int
	errBegin, errWrite, errCommit, errAbort error
}

func (s *sink) Begin() error                            { s.begin++; return s.errBegin }
func (s *sink) Write(_ uint8, _ uint32, _ []byte) error { s.writes++; return s.errWrite }
func (s *sink) Commit() error                           { s.commits++; return s.errCommit }
func (s *sink) Abort() error                            { s.aborts++; return s.errAbort }

func fixture(t *testing.T) (*Receiver, *sink, Intent) {
	t.Helper()
	s := new(sink)
	r, err := New(Config{Epoch: 7, Width: 8, Height: 2, Passes: 2,
		MaxChunk: 2, Idle: time.Second, Total: 3 * time.Second}, s)
	if err != nil {
		t.Fatal(err)
	}
	return r, s, Intent{Epoch: 7, ID: 1, Digest: sha256.Sum256([]byte{128, 1})}
}

func TestCompleteAndLostReply(t *testing.T) {
	r, s, intent := fixture(t)
	if err := r.Begin(intent, 0); err != nil {
		t.Fatal(err)
	}
	for pass := uint8(0); pass < 2; pass++ {
		if err := r.Write(intent, pass, 0, []byte{128, 1}, 0); err != nil {
			t.Fatal(err)
		}
	}
	assertReady(t, r, s)
	if err := r.Commit(intent, 0); err != nil {
		t.Fatal(err)
	}
	if err := r.Commit(intent, 0); err != nil {
		t.Fatal(err)
	}
	if err := r.Begin(intent, 0); err != nil {
		t.Fatal(err)
	}
	if s.commits != 1 || s.begin != 1 || r.Status().State != Complete {
		t.Fatal("replayed refresh")
	}
}

func assertReady(t *testing.T, r *Receiver, s *sink) {
	t.Helper()
	if s.commits != 0 || r.Status().State != Ready {
		t.Fatal("premature refresh")
	}
}

func TestCorruptSecondPassNeverRefreshes(t *testing.T) {
	r, s, intent := fixture(t)
	if err := r.Begin(intent, 0); err != nil {
		t.Fatal(err)
	}
	if err := r.Write(intent, 0, 0, []byte{128, 1}, 0); err != nil {
		t.Fatal(err)
	}
	if err := r.Write(intent, 1, 0, []byte{128, 0}, 0); !errors.Is(err, ErrDigest) {
		t.Fatal(err)
	}
	if err := r.Commit(intent, 0); !errors.Is(err, ErrState) {
		t.Fatal(err)
	}
	if s.commits != 0 || s.aborts != 1 || r.Status().State != Failed {
		t.Fatal("bad recovery")
	}
}
