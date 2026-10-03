package refreshstamp

import (
	"errors"
	"testing"
	"time"
)

func TestDiscardUnsentPreservesConfirmationButNeverReusesID(t *testing.T) {
	tracker, _ := New(time.UTC)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	label, err := tracker.Begin(1, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := tracker.Complete(1, now); err != nil {
		t.Fatal(err)
	}
	proof := tracker.Confirmed()
	if _, err := tracker.Begin(2, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := tracker.DiscardUnsent(1); !errors.Is(err, ErrCycle) {
		t.Fatal(err)
	}
	if err := tracker.DiscardUnsent(2); err != nil {
		t.Fatal(err)
	}
	retained, err := tracker.ForPartial()
	if err != nil || retained != label || tracker.Confirmed() != proof {
		t.Fatal(retained, err, tracker.Confirmed())
	}
	checkDiscardedCycle(t, tracker, now)
}

func checkDiscardedCycle(t *testing.T, tracker *Tracker, now time.Time) {
	t.Helper()
	if err := tracker.Complete(2, now.Add(time.Minute)); !errors.Is(err, ErrCycle) {
		t.Fatal(err)
	}
	if _, err := tracker.Begin(2, now.Add(time.Minute)); !errors.Is(err, ErrCycle) {
		t.Fatal(err)
	}
}

func TestDiscardUnsentCannotInventConfirmation(t *testing.T) {
	tracker, _ := New(time.UTC)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	if _, err := tracker.Begin(1, now); err != nil {
		t.Fatal(err)
	}
	if err := tracker.DiscardUnsent(0); !errors.Is(err, ErrCycle) {
		t.Fatal(err)
	}
	if err := tracker.DiscardUnsent(1); err != nil {
		t.Fatal(err)
	}
	if _, err := tracker.ForPartial(); !errors.Is(err, ErrUnconfirmed) {
		t.Fatal(err)
	}
}
