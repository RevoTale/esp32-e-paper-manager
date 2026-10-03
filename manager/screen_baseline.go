package manager

import (
	"bytes"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

// samePixels compares final canonical output, never HTML/DOM hashes. The fixed
// Screen size/stride was validated before this check. Caller holds mu.
func (s *Screen) samePixels(frame display.Frame) bool {
	return s.baseline != nil && bytes.Equal(s.baseline, frame.Bytes())
}

// InvalidatePixels fences retained evidence after device reboot, session change
// or explicit full resynchronization. During a leased render/delivery the owner
// must finish through Resolve(false), not invalidate underneath another writer.
// Delivered remains historical; Confirmed is the newest pixel-equivalent scene,
// not a claim that a new protocol ACK was received for a suppressed update.
func (s *Screen) InvalidatePixels() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.InFlight != 0 {
		return ErrConflict
	}
	s.baseline, s.state.Confirmed = nil, 0
	s.invalidateStamp()
	return nil
}
