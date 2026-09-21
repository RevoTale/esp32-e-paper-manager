package refreshstamp

import (
	"errors"
	"testing"
	"time"
)

func TestInvalidConfiguration(t *testing.T) {
	if _, err := New(nil); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	if _, err := new(Tracker).Begin(1, time.Time{}); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	if (Label{}).Text() != "" {
		t.Fatal("zero label exposes a timestamp")
	}
}

func TestInvalidCycleIdentity(t *testing.T) {
	tracker := testTracker(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := tracker.Begin(0, start); !errors.Is(err, ErrCycle) {
		t.Fatal(err)
	}
	if _, err := tracker.Begin(1, start); err != nil {
		t.Fatal(err)
	}
	for _, id := range []CycleID{1, 2} {
		if _, err := tracker.Begin(id, start); !errors.Is(err, ErrCycle) {
			t.Fatalf("parallel begin: %v", err)
		}
	}
	for _, id := range []CycleID{0, 2} {
		if err := tracker.Abort(id); !errors.Is(err, ErrCycle) {
			t.Fatalf("wrong abort: %v", err)
		}
		if err := tracker.Complete(id, start); !errors.Is(err, ErrCycle) {
			t.Fatalf("wrong completion: %v", err)
		}
	}
}

func TestInvalidTimesCannotConfirmCycle(t *testing.T) {
	tracker := testTracker(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	invalid := []time.Time{time.Time{}, time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}
	for _, instant := range invalid {
		if _, err := tracker.Begin(1, instant); !errors.Is(err, ErrTime) {
			t.Fatalf("invalid begin: %v", err)
		}
	}
	if _, err := tracker.Begin(1, start); err != nil {
		t.Fatal(err)
	}
	for _, instant := range append(invalid, start.Add(-time.Second)) {
		if err := tracker.Complete(1, instant); !errors.Is(err, ErrTime) {
			t.Fatalf("invalid completion: %v", err)
		}
	}
	if !tracker.Confirmed().Started.IsZero() {
		t.Fatal("invalid time confirmed cycle")
	}
	if err := tracker.Complete(1, start.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
}

func TestUnavailableAndBackwardClockPreservesConfirmedLabel(t *testing.T) {
	tracker := testTracker(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := tracker.Begin(1, start); err != nil {
		t.Fatal(err)
	}
	if err := tracker.Complete(1, start.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	before, err := tracker.ForPartial()
	if err != nil {
		t.Fatal(err)
	}
	for _, instant := range []time.Time{time.Time{}, start} {
		if _, err := tracker.Begin(2, instant); !errors.Is(err, ErrTime) {
			t.Fatalf("unavailable/backwards clock accepted: %v", err)
		}
	}
	if after, err := tracker.ForPartial(); err != nil || after != before {
		t.Fatalf("invalid begin changed confirmed label: %v", err)
	}
}

func TestInvalidationFencesDelayedCompletionAndRecovers(t *testing.T) {
	tracker := testTracker(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for id := CycleID(1); id <= 2; id++ {
		if _, err := tracker.Begin(id, start); err != nil {
			t.Fatal(err)
		}
		tracker.Invalidate()
		if err := tracker.Complete(id, start); !errors.Is(err, ErrCycle) {
			t.Fatalf("late completion after disconnect: %v", err)
		}
	}
	if _, err := tracker.ForPartial(); !errors.Is(err, ErrUnconfirmed) {
		t.Fatal(err)
	}
	label, err := tracker.Begin(3, start)
	if err != nil {
		t.Fatal(err)
	}
	if err := tracker.Complete(3, start.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if got, err := tracker.ForPartial(); err != nil || got != label {
		t.Fatalf("full resync did not recover: %v", err)
	}
}

func TestTimezoneDSTAndRepeatedLocalHourKeepUTCIdentity(t *testing.T) {
	tracker := testTracker(t)
	// Kyiv's repeated autumn hour has two UTC instants, never one cycle identity.
	for index, hour := range []int{0, 1} {
		start := time.Date(2026, 10, 25, hour, 30, 0, 0, time.UTC)
		id := CycleID(index + 1)
		label, err := tracker.Begin(id, start)
		if err != nil || label.Text() != "2026-10-25 03:30" {
			t.Fatalf("DST label = %q, %v", label.Text(), err)
		}
		if err := tracker.Complete(id, start.Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		if !tracker.Confirmed().Started.Equal(start) {
			t.Fatal("local duplicate hour lost UTC identity")
		}
	}
}

func TestLocalYearBoundsAndCycleExhaustion(t *testing.T) {
	for _, offset := range []int{-3600, 3600} {
		tracker, err := New(time.FixedZone("test", offset))
		if err != nil {
			t.Fatal(err)
		}
		instant := time.Date(9999, 12, 31, 23, 59, 0, 0, time.UTC)
		if offset < 0 {
			instant = time.Date(1, 1, 1, 0, 1, 0, 0, time.UTC)
		}
		if _, err := tracker.Begin(1, instant); !errors.Is(err, ErrTime) {
			t.Fatalf("out-of-range local year accepted: %v", err)
		}
	}
	tracker := testTracker(t)
	instant := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := tracker.Begin(^CycleID(0), instant); err != nil {
		t.Fatal(err)
	}
	if err := tracker.Complete(^CycleID(0), instant); err != nil {
		t.Fatal(err)
	}
	if _, err := tracker.Begin(0, instant); !errors.Is(err, ErrCycle) {
		t.Fatalf("ID wrapped: %v", err)
	}
}
