package manager

import (
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

// NewScreenPumpWithPolicy opts into the negotiated EPS2 extension. The legacy
// constructor preserves Pico's fixed cadence and rejects urgent HTTP requests.
func NewScreenPumpWithPolicy(screen *Screen, sender ScreenSender, policy refreshpolicy.Policy) (*ScreenPump, error) {
	p, err := NewScreenPump(screen, sender, policy.Normal)
	if err != nil {
		return nil, err
	}
	advanced, ok := sender.(screendelivery.PolicySender)
	if !ok {
		return nil, ErrConfiguration
	}
	if err := advanced.ConfigureRefresh(policy); err != nil {
		return nil, err
	}
	p.urgent = policy.Urgent
	screen.refreshEnabled = true
	return p, nil
}

func (s *Screen) pendingUrgent() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cycles != nil && (s.cycles.resync || s.maintenanceDue(s.elapsed())) {
		return false
	}
	return s.options.Priority == refreshpolicy.Urgent
}

func (r *screenRecovery) postponeReadiness(state screendelivery.Readiness) {
	now := r.pump.now()
	elapsed := r.pump.screen.elapsed() - r.started
	r.cooldown = max(0, r.cooldown-elapsed, state.NotBefore.Sub(now))
	urgent := state.UrgentNotBefore
	if urgent.IsZero() {
		urgent = state.NotBefore
	}
	r.urgentCooldown = max(0, r.urgentCooldown-elapsed, urgent.Sub(now))
	r.started = r.pump.screen.elapsed()
}

func (r *screenRecovery) completed() {
	r.started = r.pump.screen.elapsed()
	r.cooldown = r.pump.interval
	r.urgentCooldown = r.pump.interval
	if r.pump.urgent > 0 {
		r.urgentCooldown = r.pump.urgent
	}
}

func (r *screenRecovery) urgentRemaining() time.Duration {
	return max(0, r.urgentCooldown-(r.pump.screen.elapsed()-r.started))
}
