package manager

import (
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
)

// takeScene serializes ordinary queue leases and explicit full-cycle leases.
// Due maintenance bypasses author grouping, never the caller's availability /
// device cooldown. Its pending queue entry is subsequently consumed as a no-op.
func (s *Screen) takeScene(now time.Duration, available bool) (renderbatch.Revision, bool, error) {
	_, pending, err := s.queue.Wait(now)
	if err != nil || !available || s.state.InFlight != 0 {
		return 0, false, err
	}
	if s.cycles != nil && (s.cycles.resync || s.maintenanceDue(now)) {
		return s.takeFullCycle(pending)
	}
	rev, err := s.queue.Take(now, available)
	if rev != 0 && s.cycles != nil && rev == s.cycles.rejected {
		// A forced maintenance render already diagnosed this exact scene.
		return 0, false, s.queue.Release(rev)
	}
	return rev, false, err
}

func (s *Screen) takeFullCycle(pending bool) (renderbatch.Revision, bool, error) {
	c := s.cycles
	if s.state.Current == 0 {
		c.resync = false
		return 0, false, nil
	}
	c.forced, c.refresh = true, true
	resync := c.resync
	c.resync = false
	if resync || (pending && s.state.Current != c.rejected) {
		return s.state.Current, false, nil
	}
	// Only an explicitly rejected latest scene permits maintenance of the older
	// known-good baseline; its logical revision and diagnostic remain unchanged.
	return s.state.Confirmed, true, nil
}

func (s *Screen) prepareBaseline() (ScreenDelivery, error) {
	frame, err := display.NewFrame(s.size, (s.size.Width+7)/8, append([]byte(nil), s.baseline...))
	if err != nil {
		return s.rejectRender(s.state.InFlight, err)
	}
	s.candidate, s.prepared = frame, true
	return s.prepareDelivery()
}

func (s *Screen) refreshLease() bool {
	return s.cycles != nil && s.cycles.refresh
}

func (s *Screen) maintenanceDue(now time.Duration) bool {
	return s.baseline != nil && s.cycles != nil && now-s.cycles.completed >= s.cycles.interval
}

// wakeDelay merges author grouping and safety maintenance on one timer. Elapsed
// subtraction (not adding a deadline) avoids duration overflow and catch-up bursts.
func (s *Screen) wakeDelay(now time.Duration) (time.Duration, bool, error) {
	delay, pending, err := s.queue.Wait(now)
	if err != nil || s.state.InFlight != 0 || s.cycles == nil {
		return delay, pending && s.state.InFlight == 0, err
	}
	if s.cycles.resync && s.state.Current != 0 {
		return 0, true, nil
	}
	if s.baseline != nil {
		maintenance := max(0, s.cycles.interval-(now-s.cycles.completed))
		if !pending || maintenance < delay {
			delay = maintenance
		}
		pending = true
	}
	return delay, pending, nil
}

// forceLatest is called by the sole pump after transport reconciliation, never
// by network callbacks. It schedules one fresh full scene without changing ETag.
func (s *Screen) forceLatest() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cycles == nil || s.state.InFlight != 0 {
		return ErrConflict
	}
	s.cycles.resync = true
	s.notify()
	return nil
}
