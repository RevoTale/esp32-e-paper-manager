package manager

import (
	"context"
	"errors"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

// ScreenSender must return nil only for a matching terminal protocol success.
// Any other result is ambiguous: it must stop using the borrowed frame before
// returning. Implementations bound I/O and respect cancellation. No auto-retry.
type ScreenSender = screendelivery.Sender

// ScreenPump is the sole live renderer/delivery owner for a Screen. It does not
// poll, spawn a worker per request, or interpret errors as permission to retry.
type ScreenPump struct {
	screen   *Screen
	sender   ScreenSender
	interval time.Duration
	urgent   time.Duration
	sleep    func(context.Context, <-chan struct{}, <-chan struct{}, time.Duration, bool) error
	now      Clock
}

func NewScreenPump(screen *Screen, sender ScreenSender, interval time.Duration) (*ScreenPump, error) {
	if screen == nil || sender == nil || interval <= 0 {
		return nil, ErrConfiguration
	}
	return &ScreenPump{screen: screen, sender: sender, interval: interval, sleep: waitScreenChange, now: time.Now}, nil
}

// Run starts with a conservative cooldown because the last physical refresh is
// unknown after process restart. All deliveries here are full frames. Restart
// only after explicit transport resynchronization; this is not a retry loop.
func (p *ScreenPump) Run(ctx context.Context) (runErr error) {
	if !p.screen.pumping.CompareAndSwap(false, true) {
		return ErrConflict
	}
	defer p.screen.pumping.Store(false)
	r := screenRecovery{pump: p, started: p.screen.elapsed(), cooldown: p.interval, urgentCooldown: p.interval}
	r.transport, _ = p.sender.(screendelivery.RecoveringSender)
	defer func() { runErr = errors.Join(runErr, r.resolve(false)) }()
	for {
		if err := r.wait(ctx); err != nil {
			return err
		}
		if err := r.ready(ctx); err != nil {
			return err
		}
		if r.remaining() > 0 {
			continue
		}
		if err := r.deliver(ctx); err != nil {
			return err
		}
	}
}

func (r *screenRecovery) wait(ctx context.Context) error {
	p := r.pump
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		delay, pending, err := p.screen.waitReady()
		if err != nil {
			return err
		}
		if pending {
			delay = max(delay, r.remaining())
			if delay <= 0 {
				return nil
			}
		}
		if err := p.sleep(ctx, p.screen.wake, r.changed(), delay, pending); err != nil {
			return err
		}
		// Inspect a peer arrival even without queued author or maintenance work.
		if r.transport != nil {
			return nil
		}
	}
}

func (s *Screen) waitReady() (time.Duration, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.wakeDelay(s.elapsed())
}

func waitScreen(ctx context.Context, wake <-chan struct{}, delay time.Duration, pending bool) error {
	return waitScreenChange(ctx, wake, nil, delay, pending)
}

func waitScreenChange(ctx context.Context, wake, changed <-chan struct{}, delay time.Duration, pending bool) error {
	var tick <-chan time.Time
	if pending {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		tick = timer.C
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wake:
		return nil
	case _, open := <-changed:
		if !open {
			return ErrConfiguration
		}
		return nil
	case <-tick:
		return nil
	}
}
