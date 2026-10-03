package streamsession

import (
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestCredentialFenceRejectsOldAndUnclaimedLease(t *testing.T) {
	s, tx := fencedFixture(t)
	if s.Cooldown() != time.Second || s.Epoch().Generation != 2 {
		t.Fatal(s.Cooldown(), s.Epoch())
	}
	if _, err := s.Bind(tx.Lease); !errors.Is(err, ErrLease) {
		t.Fatal(err)
	}
	if _, err := s.Bind(Lease{Epoch: s.Epoch()}); !errors.Is(err, ErrLease) {
		t.Fatal("unclaimed epoch adopted", err)
	}
	lease, err := s.Acquire(s.Epoch(), [16]byte{7})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := s.Bind(lease)
	if err != nil {
		t.Fatal(err)
	}
	tx.Lease = lease
	result, err := s.Query(binding, tx)
	if err != nil || result.Status.State != streamrx.Idle || result.CurrentImage {
		t.Fatal(result, err)
	}
}

func fencedFixture(t *testing.T) (*Session, Transaction) {
	t.Helper()
	s, _, tx := fixture(t)
	ready(t, s, tx, time.Second)
	if err := s.Commit(s.writer, tx, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := s.Disconnect(s.writer); err != nil {
		t.Fatal(err)
	}
	if err := s.Fence(); err != nil {
		t.Fatal(err)
	}
	return s, tx
}

func TestFenceCannotInterruptBoundOwnerOrWrap(t *testing.T) {
	s, _, _ := fixture(t)
	if err := s.Fence(); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	if err := s.Disconnect(s.writer); err != nil {
		t.Fatal(err)
	}
	s.lease.Generation = ^uint64(0)
	if err := s.Fence(); !errors.Is(err, ErrLease) {
		t.Fatal(err)
	}
}
