package streamrx

import (
	"crypto/sha256"
	"errors"
	"testing"
	"time"
)

func TestInvalidChunksAndEarlyCommitAbort(t *testing.T) {
	for _, bad := range []struct {
		pass   uint8
		offset uint32
		data   []byte
	}{
		{1, 0, []byte{128}}, {0, 1, []byte{128}}, {0, 0, nil},
		{0, 0, []byte{1, 2, 3}}, {0, ^uint32(0), []byte{128}},
	} {
		r, s, intent := fixture(t)
		if err := r.Begin(intent, 0); err != nil {
			t.Fatal(err)
		}
		if err := r.Write(intent, bad.pass, bad.offset, bad.data, 0); !errors.Is(err, ErrChunk) {
			t.Fatal(err)
		}
		if s.writes != 0 || s.aborts != 1 || s.commits != 0 {
			t.Fatal("invalid chunk reached sink")
		}
	}
}

func TestEarlyCommitAbort(t *testing.T) {
	r, s, intent := fixture(t)
	if err := r.Begin(intent, 0); err != nil {
		t.Fatal(err)
	}
	if err := r.Commit(intent, 0); !errors.Is(err, ErrState) {
		t.Fatal(err)
	}
	if s.aborts != 1 || s.commits != 0 {
		t.Fatal("early commit")
	}
}

func TestAllIncompletePrefixesNeverRefresh(t *testing.T) {
	for prefix := 0; prefix < 4; prefix++ {
		r, s, intent := fixture(t)
		if err := r.Begin(intent, 0); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < prefix; i++ {
			data := []byte{128, 1}
			if err := r.Write(intent, uint8(i/2), uint32(i%2), data[i%2:i%2+1], 0); err != nil {
				t.Fatal(err)
			}
		}
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
		if s.commits != 0 || s.aborts != 1 {
			t.Fatal("disconnect refreshed")
		}
		if err := r.Begin(intent, 0); !errors.Is(err, ErrState) {
			t.Fatal(err)
		}
	}
}

func TestDeadlinesAndClockRegression(t *testing.T) {
	for _, now := range []time.Duration{-1, time.Second, 3 * time.Second} {
		r, s, intent := fixture(t)
		if err := r.Begin(intent, 0); err != nil {
			t.Fatal(err)
		}
		if err := r.Tick(now); !errors.Is(err, ErrTimeout) {
			t.Fatal(err)
		}
		if s.aborts != 1 || r.Status().State != Failed {
			t.Fatal("deadline did not abort")
		}
	}
}

func TestProgressDoesNotExtendTotalDeadline(t *testing.T) {
	r, s, intent := fixture(t)
	if err := r.Begin(intent, 0); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		data := []byte{128, 1}
		if err := r.Write(intent, uint8(i/2), uint32(i%2), data[i%2:i%2+1], time.Duration(i)*800*time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Commit(intent, 3*time.Second); !errors.Is(err, ErrTimeout) {
		t.Fatal(err)
	}
	if s.commits != 0 {
		t.Fatal("progress extended total deadline")
	}
}

func TestSinkFailuresAndRecovery(t *testing.T) {
	broken := errors.New("sink failure")
	for stage := 0; stage < 3; stage++ {
		r, s, intent := fixture(t)
		s.errAbort = broken
		switch stage {
		case 0:
			s.errBegin = broken
		case 1:
			s.errWrite = broken
		case 2:
			s.errCommit = broken
		}
		err := transfer(r, intent)
		assertFailed(t, r, s, err, broken)
		if err = r.Begin(intent, 0); !errors.Is(err, ErrState) {
			t.Fatal("failed ID reused")
		}
		s.errBegin = nil
		intent.ID++
		if err = r.Begin(intent, 0); err != nil {
			t.Fatal(err)
		}
		if err = r.Close(); !errors.Is(err, broken) {
			t.Fatal(err)
		}
	}
}

func assertFailed(t *testing.T, r *Receiver, s *sink, err, broken error) {
	t.Helper()
	if !errors.Is(err, broken) || s.aborts != 1 || r.Status().State != Failed {
		t.Fatal(err, r.Status())
	}
}

func transfer(r *Receiver, intent Intent) error {
	if err := r.Begin(intent, 0); err != nil {
		return err
	}
	for pass := uint8(0); pass < 2; pass++ {
		if err := r.Write(intent, pass, 0, []byte{128, 1}, 0); err != nil {
			return err
		}
	}
	return r.Commit(intent, 0)
}

func TestNonByteWidthPadding(t *testing.T) {
	for _, data := range [][]byte{{128, 128}, {128, 129}} {
		s := new(sink)
		r, err := New(Config{Epoch: 1, Width: 9, Height: 1, Passes: 1, MaxChunk: 1, Idle: time.Second, Total: time.Second}, s)
		if err != nil {
			t.Fatal(err)
		}
		intent := Intent{Epoch: 1, ID: 1, Digest: sha256.Sum256(data)}
		if err = r.Begin(intent, 0); err != nil {
			t.Fatal(err)
		}
		if err = r.Write(intent, 0, 0, data[:1], 0); err != nil {
			t.Fatal(err)
		}
		err = r.Write(intent, 0, 1, data[1:], 0)
		if (err == nil) != (data[1] == 128) {
			t.Fatal(err)
		}
	}
}
