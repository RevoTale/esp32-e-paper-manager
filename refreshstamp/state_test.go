package refreshstamp

import (
	"errors"
	"testing"
	"time"
)

func testTracker(t *testing.T) *Tracker {
	t.Helper()
	zone, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		t.Fatal(err)
	}
	tracker, err := New(zone)
	if err != nil {
		t.Fatal(err)
	}
	return tracker
}

func TestBeginStagesCandidateWithoutConfirming(t *testing.T) {
	tracker := testTracker(t)
	start := time.Date(2026, 9, 6, 12, 34, 0, 0, time.UTC)
	label, err := tracker.Begin(1, start)
	if err != nil || label.Text() != "2026-09-06 15:34" {
		t.Fatalf("candidate = %q, %v", label.Text(), err)
	}
	if !tracker.Confirmed().Started.IsZero() {
		t.Fatal("begin confirmed an uncompleted cycle")
	}
	if _, err := tracker.ForPartial(); !errors.Is(err, ErrUnconfirmed) {
		t.Fatalf("partial before confirmation: %v", err)
	}
}

func TestFullCycleConfirmsOnlyMatchingCompletion(t *testing.T) {
	tracker := testTracker(t)
	start := time.Date(2026, 9, 6, 12, 34, 0, 0, time.UTC)
	label, err := tracker.Begin(1, start)
	if err != nil {
		t.Fatal(err)
	}
	if err := tracker.Complete(2, start.Add(time.Second)); !errors.Is(err, ErrCycle) {
		t.Fatalf("wrong cycle: %v", err)
	}
	finish := start.Add(5 * time.Second)
	if err := tracker.Complete(1, finish); err != nil {
		t.Fatal(err)
	}
	got := tracker.Confirmed()
	if got.Cycle != 1 || !got.Started.Equal(start) || !got.Completed.Equal(finish) {
		t.Fatalf("confirmation = %+v", got)
	}
	partial, err := tracker.ForPartial()
	if err != nil || partial != label {
		t.Fatalf("partial label changed: %+v, %v", partial, err)
	}
	if err := tracker.Complete(1, finish); !errors.Is(err, ErrCycle) {
		t.Fatalf("duplicate completion: %v", err)
	}
}

func TestAbortPreservesHistoryButRequiresFullResync(t *testing.T) {
	tracker := testTracker(t)
	start := time.Date(2026, 9, 6, 12, 34, 0, 0, time.UTC)
	if _, err := tracker.Begin(1, start); err != nil {
		t.Fatal(err)
	}
	if err := tracker.Complete(1, start.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	before := tracker.Confirmed()
	if _, err := tracker.Begin(2, start.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := tracker.Abort(2); err != nil {
		t.Fatal(err)
	}
	if tracker.Confirmed() != before {
		t.Fatal("failure advanced or deleted historical confirmation")
	}
	if _, err := tracker.ForPartial(); !errors.Is(err, ErrUnconfirmed) {
		t.Fatalf("partial after ambiguous failure: %v", err)
	}
	if _, err := tracker.Begin(2, start.Add(time.Minute)); !errors.Is(err, ErrCycle) {
		t.Fatalf("reused ID: %v", err)
	}
}
