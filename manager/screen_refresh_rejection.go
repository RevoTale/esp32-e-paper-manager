package manager

import (
	"errors"

	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

// ErrPartialUnavailable rejects one scene's requested refresh mode before I/O.
// It is not a renderer failure, device failure or permission to silently send full.
var ErrPartialUnavailable = errors.New("manager: partial refresh unavailable")

type RefreshRejectionReason string

const (
	RefreshRequiresFull RefreshRejectionReason = "full-refresh-required"
	RefreshRegionLimit  RefreshRejectionReason = "region-budget"
)

// ScreenRefreshFailure contains only bounded metadata, never document contents.
type ScreenRefreshFailure struct {
	Revision renderbatch.Revision   `json:"revision"`
	Reason   RefreshRejectionReason `json:"reason"`
}

func (s *Screen) rejectPartial(rev renderbatch.Revision, cause error) (ScreenDelivery, error) {
	_, err := s.rejectRender(rev, errors.Join(ErrPartialUnavailable, cause))
	if !errors.Is(err, ErrPartialUnavailable) || rev != s.state.Current {
		return ScreenDelivery{}, err
	}
	reason := RefreshRequiresFull
	if errors.Is(cause, screendelivery.ErrRegionBudget) {
		reason = RefreshRegionLimit
	}
	s.cycles.rejected = rev
	s.state.RefreshFailure = &ScreenRefreshFailure{Revision: rev, Reason: reason}
	return ScreenDelivery{}, err
}
