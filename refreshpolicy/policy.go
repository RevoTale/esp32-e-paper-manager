// Package refreshpolicy separates operator cadence from panel safety.
// A zero wait is permission to schedule, never permission to interrupt BUSY.
package refreshpolicy

import (
	"errors"
	"time"
)

var ErrPolicy = errors.New("refresh policy: invalid metadata or interval")

type Priority uint8

const (
	Normal Priority = iota
	Urgent
)

type Mode uint8

const (
	Auto Mode = iota
	Partial
	Full
)

// Options is bound to one complete scene revision, not to individual chunks.
type Options struct {
	Priority Priority
	Mode     Mode
}

func Parse(priority, mode string) (Options, error) {
	var o Options
	switch priority {
	case "", "normal":
	case "urgent":
		o.Priority = Urgent
	default:
		return Options{}, ErrPolicy
	}
	switch mode {
	case "", "auto":
	case "partial":
		o.Mode = Partial
	case "full":
		o.Mode = Full
	default:
		return Options{}, ErrPolicy
	}
	return o, nil
}

// Policy intervals are operator budgets, not manufacturer safety guarantees.
// Both must be positive so urgent cannot accidentally mean unlimited refresh.
type Policy struct {
	Normal time.Duration
	Urgent time.Duration
}

func (p Policy) Wait(priority Priority, elapsed time.Duration) (time.Duration, error) {
	if p.Normal <= 0 || p.Urgent <= 0 || elapsed < 0 || priority > Urgent {
		return 0, ErrPolicy
	}
	interval := p.Normal
	if priority == Urgent {
		interval = p.Urgent
	}
	return max(0, interval-elapsed), nil
}
