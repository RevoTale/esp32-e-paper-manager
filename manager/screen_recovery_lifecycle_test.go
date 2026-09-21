package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func recoveryPump(t *testing.T, sender *recoveringSender) (*ScreenPump, *Screen, *cycleClock) {
	t.Helper()
	s, clock := cycleFixture(t, renderbatch.Policy{})
	p, err := NewScreenPump(s, sender, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	p.now = func() time.Time { return time.Unix(0, 0).Add(clock.tick) }
	p.sleep = func(ctx context.Context, wake, changed <-chan struct{}, delay time.Duration, pending bool) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		select {
		case <-changed:
			return nil
		default:
		}
		select {
		case <-wake:
			return nil
		default:
		}
		if !pending {
			return errors.New("unexpected idle wait")
		}
		clock.advance(delay)
		return nil
	}
	return p, s, clock
}

func TestRecoveryUnconfirmedRendersFreshLatestWithoutNewAuthorRevision(t *testing.T) {
	for _, outcome := range []screendelivery.Outcome{screendelivery.NoPending, screendelivery.PendingUnconfirmed} {
		t.Run(string(rune('0'+outcome)), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sender := &recoveringSender{}
			p, s, clock := recoveryPump(t, sender)
			submitCycle(t, s, clock, "first")
			sends, recovered := 0, false
			sender.ready = func(context.Context) (screendelivery.Readiness, error) {
				if sends == 1 && !recovered {
					recovered = true
					clock.advance(time.Minute)
					return screendelivery.Readiness{Pending: outcome}, nil
				}
				return screendelivery.Readiness{}, nil
			}
			sender.send = func(context.Context, display.Frame) error {
				sends++
				if sends == 1 {
					submitCycle(t, s, clock, "latest")
					return screendelivery.ErrTransportLost
				}
				cancel()
				return nil
			}
			if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			status := s.Status()
			if got := [4]uint64{uint64(sends), uint64(status.Current), uint64(status.Confirmed), uint64(status.FullRefresh.Cycle)}; got != [4]uint64{2, 2, 2, 2} {
				t.Fatal(sends, status)
			}
			if status.FullRefresh.Started != clock.wall {
				t.Fatal("old offline stamp reused", status.FullRefresh)
			}
		})
	}
}

func TestRecoveryIdlePeerChangeDoesNotRedrawUnlessResync(t *testing.T) {
	for _, resync := range []bool{false, true} {
		t.Run(map[bool]string{false: "same-boot", true: "new-boot"}[resync], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sender := &recoveringSender{changed: make(chan struct{}, 1)}
			p, s, clock := recoveryPump(t, sender)
			submitCycle(t, s, clock, "cached")
			sends, peerChanged := 0, false
			sender.ready = func(context.Context) (screendelivery.Readiness, error) {
				if peerChanged {
					peerChanged = false
					if resync {
						return screendelivery.Readiness{}, screendelivery.ErrTransportResync
					}
					cancel()
				}
				return screendelivery.Readiness{}, nil
			}
			sender.send = func(context.Context, display.Frame) error {
				sends++
				if sends == 1 {
					peerChanged = true
					sender.changed <- struct{}{}
				} else {
					cancel()
				}
				return nil
			}
			if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			want := 1
			if resync {
				want++
			}
			if sends != want || s.Status().Current != 1 || sender.resets != want-1 {
				t.Fatal(sends, sender.resets, s.Status())
			}
		})
	}
}

func TestRecoveryCooldownPrecedesLatestSceneAndTimestamp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sender := &recoveringSender{}
	p, s, clock := recoveryPump(t, sender)
	submitCycle(t, s, clock, "old")
	notBefore := p.now().Add(time.Hour)
	sender.ready = func(context.Context) (screendelivery.Readiness, error) {
		return screendelivery.Readiness{NotBefore: notBefore}, nil
	}
	baseSleep := p.sleep
	submitted := false
	p.sleep = func(ctx context.Context, wake, changed <-chan struct{}, delay time.Duration, pending bool) error {
		if delay > time.Minute && !submitted {
			submitted = true
			submitCycle(t, s, clock, "latest")
		}
		return baseSleep(ctx, wake, changed, delay, pending)
	}
	sender.send = func(context.Context, display.Frame) error { cancel(); return nil }
	if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if clock.tick < time.Hour || s.Status().Confirmed != s.Status().Current || s.Status().FullRefresh.Started != clock.wall {
		t.Fatal("render/stamp preceded readiness cooldown", clock.tick, s.Status())
	}
}
