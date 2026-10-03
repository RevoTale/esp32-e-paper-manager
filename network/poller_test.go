package network

import (
	"errors"
	"testing"
	"time"
)

func TestPollManagerUSBFailureAndSuccessPolicy(t *testing.T) {
	steps, exchanges := 0, 0
	sleeps := make([]time.Duration, 0, 4)
	err := PollManager(PollHooks{
		Continue: func() bool { steps++; return steps <= 4 },
		USB:      func() bool { return steps == 1 },
		Exchange: func() error {
			exchanges++
			if exchanges == 1 {
				return errors.New("offline")
			}
			return nil
		},
		Sleep: func(duration time.Duration) { sleeps = append(sleeps, duration) }, Interval: 30 * time.Second,
	})
	if err != nil || exchanges != 3 || len(sleeps) != 4 || sleeps[0] != time.Second ||
		sleeps[1] != 2*time.Second || sleeps[2] != 30*time.Second || sleeps[3] != 30*time.Second {
		t.Fatalf("exchanges=%d sleeps=%v err=%v", exchanges, sleeps, err)
	}
}

func TestPollManagerRejectsIncompleteHooks(t *testing.T) {
	if err := PollManager(PollHooks{}); !errors.Is(err, ErrPolling) {
		t.Fatalf("error=%v", err)
	}
}
