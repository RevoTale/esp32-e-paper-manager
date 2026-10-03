package manager

import (
	"context"
	"time"
)

// Preparation owns pixels but has performed no device I/O. Keep this distinct
// from a sent transaction whose Commit reply may have been lost.
func (s *Screen) discardUnsent(d ScreenDelivery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.prepared || s.state.InFlight != d.Revision || s.state.InFlightCycle != d.Cycle {
		return ErrConflict
	}
	if s.cycles != nil && !s.cycles.partial {
		if err := s.cycles.tracker.DiscardUnsent(d.Cycle); err != nil {
			return err
		}
	}
	return s.release()
}

// A delayed full frame carries the dispatch time, not its early render time.
// The unstamped candidate remains immutable; no renderer rerun is necessary.
func (s *Screen) stampAtDispatch(d ScreenDelivery) (ScreenDelivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cycles == nil || s.cycles.partial || s.cycles.started == s.elapsed() {
		return d, nil
	}
	if err := s.cycles.tracker.DiscardUnsent(d.Cycle); err != nil {
		return ScreenDelivery{}, err
	}
	return s.prepareFullDelivery(d)
}

func (r *screenRecovery) dispatch(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// WaitReady may block while author work changes. Recheck before device I/O,
	// not just before the timer/readiness wait.
	if _, _, err := r.waitReady(); err != nil {
		return err
	}
	if !r.unsent {
		return nil
	}
	if r.remainingFor(r.pending.Options) > 0 {
		r.delayed = true
		return nil
	}
	if r.delayed {
		d, err := r.pump.screen.stampAtDispatch(r.pending)
		if err != nil {
			return err
		}
		r.pending = d
	}
	return r.send(ctx)
}

func (r *screenRecovery) waitReady() (time.Duration, bool, error) {
	if r.unsent {
		stale, maintenance := r.pump.screen.replanUnsent(r.pending)
		if !stale && !maintenance {
			return r.remainingFor(r.pending.Options), true, nil
		}
		if err := r.resolve(false); err != nil {
			return 0, false, err
		}
		if maintenance {
			if err := r.pump.screen.forceLatest(); err != nil {
				return 0, false, err
			}
		}
	}
	return r.pump.screen.waitReady()
}

func (s *Screen) replanUnsent(d ScreenDelivery) (stale, maintenance bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Only an already rejected latest scene permits maintenance of old pixels.
	retainRejected := s.refreshLease() && s.cycles.rejected == s.state.Current
	return d.Revision != s.state.Current && !retainRejected,
		d.Region != nil && s.maintenanceDue(s.elapsed())
}
