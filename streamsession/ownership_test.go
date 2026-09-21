package streamsession

import (
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestLostAcquireACKDoesNotAdoptCompetitor(t *testing.T) {
	s, _, tx := fixture(t)
	old := Epoch{Boot: tx.Boot}
	ready(t, s, tx, time.Second)
	if err := s.Commit(s.writer, tx, time.Second); err != nil {
		t.Fatal(err)
	}
	writer := s.writer
	grant, err := s.Acquire(old, tx.Claim)
	if err != nil || grant != tx.Lease || s.writer != writer || s.consumed != tx.ID {
		t.Fatal(grant, err)
	}
	if _, err = s.Acquire(old, [16]byte{2}); !errors.Is(err, ErrLease) {
		t.Fatal(err)
	}
	result, err := s.Query(writer, tx)
	if err != nil || !result.CurrentImage || s.Cooldown() != time.Second {
		t.Fatal(result, err)
	}
}

func TestRetiredClaimCannotAcquireOrBind(t *testing.T) {
	s, _, tx := fixture(t)
	old := Epoch{Boot: tx.Boot}
	if err := s.Disconnect(s.writer); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Acquire(s.Epoch(), [16]byte{2}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Acquire(old, tx.Claim); !errors.Is(err, ErrLease) {
		t.Fatal("old retry", err)
	}
	if _, err := s.Bind(tx.Lease); !errors.Is(err, ErrLease) {
		t.Fatal("old claim", err)
	}
}

func TestBindingSerialFencesLateDisconnect(t *testing.T) {
	s, sink, tx := fixture(t)
	old := s.writer
	if _, err := s.Bind(tx.Lease); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	if err := s.Disconnect(old); err != nil {
		t.Fatal(err)
	}
	current, err := s.Bind(tx.Lease)
	if err != nil || old.Serial == current.Serial {
		t.Fatal(current, err)
	}
	if err = s.Begin(current, tx, time.Second); err != nil {
		t.Fatal(err)
	}
	assertLateClose(t, s, sink, old)
	if err = s.Disconnect(current); err != nil || sink.aborts != 1 {
		t.Fatal(sink, err)
	}
}

func assertLateClose(t *testing.T, s *Session, sink *testSink, old Binding) {
	t.Helper()
	if err := s.Disconnect(old); !errors.Is(err, ErrLease) {
		t.Fatal(err)
	}
	if err := s.Abort(old); !errors.Is(err, ErrLease) {
		t.Fatal(err)
	}
	if sink.aborts != 0 || !s.active() {
		t.Fatal("late close aborted replacement")
	}
}

func TestRebootAndInvalidClaimsReject(t *testing.T) {
	s, _, tx := fixture(t)
	cases := []struct {
		epoch Epoch
		claim [16]byte
	}{
		{Epoch{Boot: [16]byte{2}}, tx.Claim},
		{s.Epoch(), [16]byte{}},
		{Epoch{Boot: tx.Boot, Generation: ^uint64(0)}, tx.Claim},
	}
	for _, test := range cases {
		if grant, err := s.Acquire(test.epoch, test.claim); !errors.Is(err, ErrLease) || grant != (Lease{}) {
			t.Fatal(grant, err)
		}
	}
	if err := s.Disconnect(s.writer); err != nil {
		t.Fatal(err)
	}
	s.serial = ^uint64(0)
	if _, err := s.Bind(tx.Lease); !errors.Is(err, ErrLease) {
		t.Fatal(err)
	}
}

func TestStaleBindingCannotMutateSession(t *testing.T) {
	s, sink, tx := fixture(t)
	bad := s.writer
	bad.Serial++
	calls := []func() error{
		func() error { return s.Begin(bad, tx, time.Second) },
		func() error { return s.Write(bad, tx, 0, 0, []byte{128}, time.Second) },
		func() error { return s.Commit(bad, tx, time.Second) },
		func() error { _, err := s.Query(bad, tx); return err },
	}
	for _, call := range calls {
		if err := call(); !errors.Is(err, ErrLease) {
			t.Fatal(err)
		}
	}
	if sink.begins+sink.writes+sink.commits+sink.aborts != 0 || s.observed != 0 {
		t.Fatal(sink)
	}
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	config := streamrx.Config{Width: 8, Height: 1, Passes: 1, MaxChunk: 1, Idle: time.Second, Total: time.Second}
	for _, minimum := range []time.Duration{0, -1} {
		if _, err := New([16]byte{1}, config, &testSink{}, minimum); !errors.Is(err, streamrx.ErrConfig) {
			t.Fatal(err)
		}
	}
	if _, err := New([16]byte{}, config, &testSink{}, time.Second); !errors.Is(err, streamrx.ErrConfig) {
		t.Fatal(err)
	}
	if _, err := New([16]byte{1}, config, nil, time.Second); !errors.Is(err, streamrx.ErrConfig) {
		t.Fatal(err)
	}
	config.Width = 0
	if _, err := New([16]byte{1}, config, &testSink{}, time.Second); !errors.Is(err, streamrx.ErrConfig) {
		t.Fatal(err)
	}
}
