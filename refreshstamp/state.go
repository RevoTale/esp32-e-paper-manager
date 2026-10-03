// Package refreshstamp owns manager-side full-refresh labels, not panel timing.
// Callers serialize access and authenticate device results before confirmation.
package refreshstamp

import (
	"errors"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/bitmapfont"
)

var (
	ErrConfiguration = errors.New("refreshstamp: timezone required")
	ErrCycle         = errors.New("refreshstamp: stale or conflicting cycle")
	ErrTime          = errors.New("refreshstamp: trusted time unavailable or invalid")
	ErrUnconfirmed   = errors.New("refreshstamp: full resynchronization required")
)

type CycleID uint64

// Label is immutable and constructed only from an explicitly supplied instant.
type Label struct {
	text  [bitmapfont.TimestampLength]byte
	valid bool
}

func (l Label) Text() string {
	if !l.valid {
		return ""
	}
	return string(l.text[:])
}

// Confirmation records historical protocol evidence, not visible acceptance.
// It remains available after invalidation; ForPartial independently guards reuse.
type Confirmation struct {
	Cycle              CycleID
	Started, Completed time.Time
}

// Tracker is scoped to one coordinator/device session. IDs never repeat within
// that session. A fresh tracker requires full resync; it cannot restore from a
// wall clock or the retained e-paper image alone. The zero value rejects Begin.
type Tracker struct {
	zone               *time.Location
	lastID             CycleID
	pending, confirmed Confirmation
	candidate, label   Label
	ready              bool
}

func New(zone *time.Location) (*Tracker, error) {
	if zone == nil {
		return nil, ErrConfiguration
	}
	return &Tracker{zone: zone}, nil
}

// Begin stages a full cycle without confirming it. The caller establishes clock
// trust; this function only validates shape/order. No implicit time.Now or UTC
// fallback. UTC normalization deliberately uses wall instants across processes:
// https://pkg.go.dev/time#hdr-Monotonic_Clocks
func (t *Tracker) Begin(id CycleID, started time.Time) (Label, error) {
	if t.zone == nil {
		return Label{}, ErrConfiguration
	}
	if id == 0 || id <= t.lastID || t.pending.Cycle != 0 {
		return Label{}, ErrCycle
	}
	started = started.UTC()
	local := started.In(t.zone)
	if !validTime(started) || local.Year() < 1 || local.Year() > 9999 ||
		started.Before(t.confirmed.Completed) {
		return Label{}, ErrTime
	}
	label := Label{valid: true}
	copy(label.text[:], local.Format("2006-01-02 15:04"))
	t.lastID, t.pending = id, Confirmation{Cycle: id, Started: started}
	t.candidate = label
	return label, nil
}

func (t *Tracker) Complete(id CycleID, completed time.Time) error {
	if id == 0 || id != t.pending.Cycle {
		return ErrCycle
	}
	completed = completed.UTC()
	if !validTime(completed) || completed.Before(t.pending.Started) {
		return ErrTime
	}
	t.confirmed, t.label = t.pending, t.candidate
	t.confirmed.Completed = completed
	t.pending, t.candidate, t.ready = Confirmation{}, Label{}, true
	return nil
}

// Abort covers failed or unknown results. No retry and no confirmed time advance.
// DiscardUnsent is deliberately separate: it requires proof of no device I/O.
func (t *Tracker) Abort(id CycleID) error {
	if id == 0 || id != t.pending.Cycle {
		return ErrCycle
	}
	t.Invalidate()
	return nil
}

// DiscardUnsent releases preparation only. The caller must prove that no
// transport operation began; a failed/unknown send MUST use Abort/Invalidate.
// The old confirmed label survives, but the discarded ID cannot be reused.
func (t *Tracker) DiscardUnsent(id CycleID) error {
	if id == 0 || id != t.pending.Cycle {
		return ErrCycle
	}
	t.pending, t.candidate = Confirmation{}, Label{}
	return nil
}

// Invalidate fences delayed completion after disconnect/reset/ambiguous state.
// Historical confirmation and used IDs remain; only a new full cycle restores reuse.
func (t *Tracker) Invalidate() {
	t.pending, t.candidate, t.ready = Confirmation{}, Label{}, false
}

func (t *Tracker) Confirmed() Confirmation { return t.confirmed }

func (t *Tracker) ForPartial() (Label, error) {
	if !t.ready || t.pending.Cycle != 0 {
		return Label{}, ErrUnconfirmed
	}
	return t.label, nil
}

func validTime(value time.Time) bool {
	return !value.IsZero() && value.Year() >= 1 && value.Year() <= 9999
}
