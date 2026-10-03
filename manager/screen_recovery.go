package manager

import (
	"context"
	"errors"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

// One pump retains one leased server frame until the same client's uncertainty
// is reconciled. Network callbacks never mutate this lease or pixel baseline.
type screenRecovery struct {
	pump                  *ScreenPump
	transport             screendelivery.RecoveringSender
	pending               ScreenDelivery
	unsent                bool
	delayed               bool
	started, cooldown     time.Duration
	urgentCooldown        time.Duration
	partialCooldown       time.Duration
	partialUrgentCooldown time.Duration
}

func (r *screenRecovery) changed() <-chan struct{} {
	if r.transport == nil {
		return nil
	}
	return r.transport.Changed()
}

func (r *screenRecovery) remaining() time.Duration {
	if r.unsent {
		return r.remainingFor(r.pending.Options)
	}
	options := r.pump.screen.pendingOptions()
	delay := r.remainingFor(options)
	if options.Mode == refreshpolicy.Auto && r.pump.partialPolicy.Normal > 0 {
		options.Mode = refreshpolicy.Partial
		delay = min(delay, r.remainingFor(options))
	}
	return delay
}

func (r *screenRecovery) postpone(delay time.Duration) {
	elapsed := r.pump.screen.elapsed() - r.started
	r.cooldown = max(0, r.cooldown-elapsed, delay)
	r.urgentCooldown = max(0, r.urgentCooldown-elapsed, delay)
	r.partialCooldown = max(0, r.partialCooldown-elapsed, delay, r.pump.partialPolicy.Normal)
	r.partialUrgentCooldown = max(0, r.partialUrgentCooldown-elapsed, delay, r.pump.partialPolicy.Urgent)
	r.started = r.pump.screen.elapsed()
}

func (r *screenRecovery) resolve(confirmed bool) error {
	d := r.pending
	if d.Revision == 0 {
		return nil
	}
	if r.unsent && confirmed {
		return ErrConflict
	}
	r.pending = ScreenDelivery{}
	r.delayed = false
	if r.unsent {
		r.unsent = false
		return r.pump.screen.discardUnsent(d)
	}
	if d.Cycle != 0 {
		return r.pump.screen.ResolveCycle(d.Cycle, confirmed)
	}
	return r.pump.screen.Resolve(d.Revision, confirmed)
}

func (r *screenRecovery) ready(ctx context.Context) error {
	if r.transport == nil {
		return nil
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		state, err := r.transport.WaitReady(ctx)
		if errors.Is(err, screendelivery.ErrTransportResync) {
			if err = r.reset(); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err = r.reconcile(state.Pending); err != nil {
			return err
		}
		r.postponeReadiness(state)
		return nil
	}
}

func (r *screenRecovery) reconcile(outcome screendelivery.Outcome) error {
	if outcome > screendelivery.PendingUnconfirmed {
		return ErrConflict
	}
	if r.pending.Revision == 0 || r.unsent {
		if outcome != screendelivery.NoPending {
			return ErrConflict
		}
		return nil
	}
	confirmed := outcome == screendelivery.PendingConfirmed
	if err := r.resolve(confirmed); err != nil {
		return err
	}
	r.postpone(r.pump.interval)
	if !confirmed {
		return r.pump.screen.forceLatest()
	}
	return nil
}

func (r *screenRecovery) reset() error {
	if err := r.resolve(false); err != nil {
		return err
	}
	if err := r.pump.screen.InvalidatePixels(); err != nil {
		return err
	}
	if err := r.pump.screen.forceLatest(); err != nil {
		return err
	}
	r.transport.ResetSession()
	r.postpone(r.pump.interval)
	return nil
}

func (r *screenRecovery) send(ctx context.Context) error {
	r.unsent = false
	var err error
	if r.pending.Region != nil {
		err = r.sendRegion(ctx)
	} else if advanced, ok := r.pump.sender.(screendelivery.PolicySender); ok && r.pump.urgent > 0 {
		err = advanced.SendWithOptions(ctx, r.pending.Frame, r.pending.Options)
	} else {
		err = r.pump.sender.Send(ctx, r.pending.Frame)
	}
	if r.transport != nil && errors.Is(err, screendelivery.ErrTransportLost) {
		r.uncertain()
		return r.ready(ctx)
	}
	if r.transport != nil && errors.Is(err, screendelivery.ErrTransportResync) {
		if err := r.reset(); err != nil {
			return err
		}
		return r.ready(ctx)
	}
	result := errors.Join(err, r.resolve(err == nil))
	if err == nil {
		r.completed()
	} else {
		r.postpone(r.pump.interval)
	}
	return result
}

func (r *screenRecovery) uncertain() {
	s := r.pump.screen
	s.mu.Lock()
	defer s.mu.Unlock()
	// The old image may already have changed. Preserve only the exact pending
	// candidate/tracker lease so a later authenticated Query can still confirm it.
	s.baseline, s.state.Confirmed, s.state.RefreshTrusted = nil, 0, false
}

func (r *screenRecovery) deliver(ctx context.Context) error {
	if r.unsent {
		return r.dispatch(ctx)
	}
	d, err := r.pump.screen.renderAt(ctx, r.pump.screen.elapsed, true)
	if errors.Is(err, ErrSuperseded) || errors.Is(err, renderdiag.ErrRejected) || errors.Is(err, ErrPartialUnavailable) {
		// Rejection/supersession is consumed; wait for explicit new author work.
		return nil
	}
	if err != nil {
		return err
	}
	if d.Revision == 0 {
		return nil
	}
	r.pending = d
	r.unsent = true
	r.delayed = false
	return r.dispatch(ctx)
}
