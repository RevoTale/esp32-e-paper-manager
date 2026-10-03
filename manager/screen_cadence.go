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

// ConfigurePartial opts into mode-aware scheduling before Run. The caller must
// serialize configuration with startup. Geometry/count limits belong to Screen;
// the transport negotiates support and enforces the same cadence on the wire.
func (p *ScreenPump) ConfigurePartial(policy refreshpolicy.Policy) error {
	if p.screen.pumping.Load() {
		return ErrConflict
	}
	if p.urgent == 0 || p.screen.cycles == nil || p.screen.cycles.partialOptions == nil {
		return ErrConfiguration
	}
	sender, ok := p.sender.(screendelivery.RegionPolicySender)
	if !ok {
		return ErrConfiguration
	}
	if err := sender.ConfigurePartial(policy); err != nil {
		return err
	}
	p.partialPolicy = policy
	p.screen.partialEnabled = true
	return nil
}

func (s *Screen) pendingOptions() refreshpolicy.Options {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cycles != nil && (s.cycles.resync || s.maintenanceDue(s.elapsed())) {
		return refreshpolicy.Options{Mode: refreshpolicy.Full}
	}
	options := s.options
	if s.cycles == nil || s.cycles.partialOptions == nil || s.baseline == nil ||
		!s.state.RefreshTrusted || s.cycles.partials >= s.cycles.partialOptions.MaxConsecutive {
		options.Mode = refreshpolicy.Full
	}
	return options
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
	r.postponePartialReadiness(state, now, elapsed)
	r.started = r.pump.screen.elapsed()
}

func (r *screenRecovery) completed() {
	r.started = r.pump.screen.elapsed()
	r.cooldown = r.pump.interval
	r.urgentCooldown = r.pump.interval
	if r.pump.urgent > 0 {
		r.urgentCooldown = r.pump.urgent
	}
	r.partialCooldown, r.partialUrgentCooldown = r.cooldown, r.urgentCooldown
	if r.pump.partialPolicy.Normal > 0 {
		r.partialCooldown = r.pump.partialPolicy.Normal
		r.partialUrgentCooldown = r.pump.partialPolicy.Urgent
	}
}

// Auto is not partial permission: only a prepared delivery can select the
// shorter lane. Geometry, baseline trust and maintenance can require full.
func (r *screenRecovery) remainingFor(options refreshpolicy.Options) time.Duration {
	normal, urgent := r.cooldown, r.urgentCooldown
	if options.Mode == refreshpolicy.Partial && r.pump.partialPolicy.Normal > 0 {
		normal, urgent = r.partialCooldown, r.partialUrgentCooldown
	}
	if options.Priority == refreshpolicy.Urgent {
		normal = urgent
	}
	return max(0, normal-(r.pump.screen.elapsed()-r.started))
}

func (r *screenRecovery) postponePartialReadiness(state screendelivery.Readiness, now time.Time, elapsed time.Duration) {
	normal, urgent := state.PartialNotBefore, state.PartialUrgentNotBefore
	if normal.IsZero() {
		normal = state.NotBefore
	}
	if urgent.IsZero() {
		urgent = normal
	}
	r.partialCooldown = max(0, r.partialCooldown-elapsed, normal.Sub(now))
	r.partialUrgentCooldown = max(0, r.partialUrgentCooldown-elapsed, urgent.Sub(now))
}
