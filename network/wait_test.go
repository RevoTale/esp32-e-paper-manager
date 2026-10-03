package network

import (
	"errors"
	"testing"
	"time"
)

func TestAsyncWaitChecksCancellationBeforeAnyOperation(t *testing.T) {
	now := time.Unix(0, 0)
	steps := 0
	w := Waiter{Now: func() time.Time { return now }, Sleep: func(d time.Duration) { now = now.Add(d) }}
	err := w.Wait(time.Second, func() (bool, error) { steps++; return false, nil }, func() bool { return true })
	if !errors.Is(err, ErrCancelled) || steps != 0 {
		t.Fatal(err, steps)
	}
	err = w.Wait(45*time.Millisecond, func() (bool, error) { steps++; return false, nil }, func() bool { return false })
	if !errors.Is(err, ErrTimeout) || steps != 3 || now != time.Unix(0, 0).Add(45*time.Millisecond) {
		t.Fatal(err, steps, now)
	}
}

func TestAsyncWaitSuccessErrorAndInvalidClock(t *testing.T) {
	now := time.Unix(0, 0)
	w := Waiter{Now: func() time.Time { return now }, Sleep: func(d time.Duration) { now = now.Add(d) }}
	never := func() bool { return false }
	if err := w.Wait(time.Second, func() (bool, error) { return true, nil }, never); err != nil {
		t.Fatal(err)
	}
	cause := errors.New("fixture")
	if err := w.Wait(time.Second, func() (bool, error) { return false, cause }, never); err != cause {
		t.Fatal(err)
	}
	w.Sleep = func(time.Duration) { now = now.Add(-time.Second) }
	if err := w.Wait(time.Second, func() (bool, error) { return false, nil }, never); err != ErrPolling {
		t.Fatal(err)
	}
	if err := (Waiter{}).Wait(time.Second, nil, nil); err != ErrPolling {
		t.Fatal(err)
	}
}
