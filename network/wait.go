package network

import (
	"errors"
	"time"
)

var (
	ErrCancelled = errors.New("network: lifetime cancelled")
	ErrTimeout   = errors.New("network: asynchronous operation deadline exceeded")
)

// Waiter polls existing asynchronous stack APIs. Check must be nonblocking;
// this cannot turn a blocking CYW Join call into a cancellable operation.
type Waiter struct {
	Now   func() time.Time
	Sleep func(time.Duration)
}

func (w Waiter) Wait(budget time.Duration, check func() (bool, error), cancelled func() bool) error {
	if !w.valid(budget, check, cancelled) {
		return ErrPolling
	}
	observed := w.Now()
	deadline := observed.Add(budget)
	for {
		if cancelled() {
			return ErrCancelled
		}
		now := w.Now()
		if now.Before(observed) {
			return ErrPolling
		}
		observed = now
		if !now.Before(deadline) {
			return ErrTimeout
		}
		ready, err := check()
		if err != nil || ready {
			return err
		}
		w.Sleep(min(20*time.Millisecond, deadline.Sub(now)))
	}
}

func (w Waiter) valid(budget time.Duration, check func() (bool, error), cancelled func() bool) bool {
	return w.Now != nil && w.Sleep != nil && check != nil && cancelled != nil && budget > 0
}
