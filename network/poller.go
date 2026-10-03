package network

import (
	"errors"
	"time"
)

var ErrPolling = errors.New("network: invalid poller configuration")

type PollHooks struct {
	Continue func() bool
	USB      func() bool
	Exchange func() error
	Sleep    func(time.Duration)
	Interval time.Duration
}

// PollManager runs the outbound-only device lifecycle. Successful polls use a
// low-frequency interval; failures use capped exponential backoff; active USB
// always wins arbitration.
func PollManager(hooks PollHooks) error {
	if hooks.Continue == nil || hooks.USB == nil || hooks.Exchange == nil || hooks.Sleep == nil || hooks.Interval <= 0 {
		return ErrPolling
	}
	failures := uint(0)
	for hooks.Continue() {
		if hooks.USB() {
			hooks.Sleep(time.Second)
			continue
		}
		if err := hooks.Exchange(); err != nil {
			hooks.Sleep(RetryDelay(failures))
			failures++
			continue
		}
		failures = 0
		hooks.Sleep(hooks.Interval)
	}
	return nil
}
