// Package renderbatch implements per-device manager batching, not panel cadence.
// It holds revision handles only; the owner retains immutable complete targets.
package renderbatch

import (
	"errors"
	"time"
)

var (
	ErrPolicy   = errors.New("renderbatch: negative duration")
	ErrClock    = errors.New("renderbatch: negative or regressing monotonic clock")
	ErrRevision = errors.New("renderbatch: stale or invalid revision")
)

type Revision uint64

// Policy uses zero for omitted settings. MaxWait alone never introduces delay.
// Parse external durations with checked conversion; do not cast float seconds.
type Policy struct {
	Debounce time.Duration
	MaxWait  time.Duration
}

// Queue is single-owner. Serialize API edits before Submit, so a new revision
// represents the whole canonical scene, not an independently rendered patch.
// No goroutines, timers, frame copies, or unbounded history are owned here.
type Queue struct {
	policy                  Policy
	latest, pending, flight Revision
	first, last, observed   time.Duration
	urgent                  bool
}

func New(policy Policy) (*Queue, error) {
	if policy.Debounce < 0 || policy.MaxWait < 0 {
		return nil, ErrPolicy
	}
	return &Queue{policy: policy}, nil
}

// Submit replaces only pending work. Revisions never wrap within this queue;
// start a new coordinator epoch before exhaustion. Stale renders cannot submit.
func (q *Queue) Submit(revision Revision, now time.Duration) error {
	if revision == 0 || revision <= q.latest {
		return ErrRevision
	}
	if err := q.clock(now); err != nil {
		return err
	}
	if q.pending == 0 {
		q.first = now
	}
	q.last, q.latest, q.pending = now, revision, revision
	q.urgent = false
	return nil
}

// SubmitUrgent bypasses grouping for this replacement only. It cannot bypass
// availability or the existing flight lease, and does not authorize refresh.
func (q *Queue) SubmitUrgent(revision Revision, now time.Duration) error {
	if err := q.Submit(revision, now); err != nil {
		return err
	}
	q.urgent = true
	return nil
}

// Wait returns the remaining grouping delay. False means no pending work.
// Caller can use a single timer and wake it on Submit, availability or Release.
// Elapsed-time subtraction avoids overflow when durations approach MaxInt64.
// Supply time.Since(serverStart), not Unix time, to preserve monotonic behavior:
// https://pkg.go.dev/time#hdr-Monotonic_Clocks
func (q *Queue) Wait(now time.Duration) (time.Duration, bool, error) {
	if err := q.clock(now); err != nil {
		return 0, false, err
	}
	if q.pending == 0 {
		return 0, false, nil
	}
	if q.urgent {
		return 0, true, nil
	}
	wait := remaining(q.policy.Debounce, now-q.last)
	if q.policy.MaxWait > 0 {
		wait = min(wait, remaining(q.policy.MaxWait, now-q.first))
	}
	return wait, true, nil
}

// Take leases the newest ready revision. Availability includes transport,
// renderer readiness and device safety. MaxWait never overrides these gates.
// Zero means nothing can dispatch. New arrivals start a separate pending window.
func (q *Queue) Take(now time.Duration, available bool) (Revision, error) {
	wait, pending, err := q.Wait(now)
	if err != nil {
		return 0, err
	}
	if !available || q.flight != 0 || !pending || wait > 0 {
		return 0, nil
	}
	q.flight, q.pending = q.pending, 0
	return q.flight, nil
}

// Release ends exactly the matching lease, without claiming display success or
// retrying it. The owner must reconcile failures/unknown ACKs before availability
// becomes true again. Confirmed frame/revision is deliberately not tracked here.
func (q *Queue) Release(revision Revision) error {
	if revision == 0 || revision != q.flight {
		return ErrRevision
	}
	q.flight = 0
	return nil
}

func (q *Queue) clock(now time.Duration) error {
	if now < 0 || now < q.observed {
		return ErrClock
	}
	q.observed = now
	return nil
}

func remaining(interval, elapsed time.Duration) time.Duration {
	if elapsed >= interval {
		return 0
	}
	return interval - elapsed
}
