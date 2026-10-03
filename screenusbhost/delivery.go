package screenusbhost

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func (s *Sender) WaitReady(ctx context.Context) (screendelivery.Readiness, error) {
	for {
		if err := ctx.Err(); err != nil {
			return screendelivery.Readiness{}, err
		}
		p, err := s.process()
		if err != nil {
			return screendelivery.Readiness{}, err
		}
		ready, err := s.readiness(ctx, p)
		if err == nil {
			return ready, nil
		}
		if closeErr := s.disconnect(); closeErr != nil {
			return ready, errors.Join(err, closeErr)
		}
		if !retryable(err) {
			return ready, translate(err)
		}
		if err = s.retry(ctx); err != nil {
			return ready, err
		}
	}
}

func (s *Sender) readiness(ctx context.Context, p worker) (screendelivery.Readiness, error) {
	select {
	case <-p.Done():
		s.bound = false
	default:
	}
	if s.bound {
		probe := s.client.Probe
		if s.rebind {
			probe = s.client.Rebind
		}
		return s.cadence.Ready(s.floor), operation(ctx, p, probe)
	}
	var ready screendelivery.Readiness
	err := operation(ctx, p, func() error { var err error; ready, err = s.connect(p); return err })
	return ready, err
}

func (s *Sender) connect(p worker) (screendelivery.Readiness, error) {
	caps, err := s.client.Connect(p)
	if err != nil {
		return screendelivery.Readiness{}, err
	}
	if int(caps.Width) != s.size.Width || int(caps.Height) != s.size.Height {
		return screendelivery.Readiness{}, ErrConfiguration
	}
	s.caps = caps
	if !s.cadence.Supports(caps) {
		return screendelivery.Readiness{}, screenclient.ErrUnsupportedRefresh
	}
	if s.floor.IsZero() {
		s.cooldown()
	}
	r := s.cadence.Ready(s.floor)
	if s.client.Pending() {
		result, err := s.client.Reconcile()
		if err != nil {
			return r, err
		}
		r.Pending = screendelivery.PendingUnconfirmed
		if result.Confirmed {
			r.Pending = screendelivery.PendingConfirmed
		}
		s.cooldown()
		outcome := r.Pending
		r = s.cadence.Ready(s.floor)
		r.Pending = outcome
	}
	s.bound = true
	return r, nil
}

func (s *Sender) Send(ctx context.Context, frame display.Frame) error {
	return s.SendWithOptions(ctx, frame, refreshpolicy.Options{})
}

func (s *Sender) SendWithOptions(ctx context.Context, frame display.Frame, options refreshpolicy.Options) error {
	return s.send(ctx, options, func() error { return s.sendFrame(frame, options) })
}

func (s *Sender) send(ctx context.Context, options refreshpolicy.Options, transmit func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !s.bound {
		return screendelivery.ErrTransportLost
	}
	if s.now().Before(s.cadence.Deadline(s.floor, options)) {
		return ErrCooldown
	}
	p, err := s.process()
	if err != nil {
		return err
	}
	err = operation(ctx, p, transmit)
	if err == nil {
		if s.cadence.Enabled() {
			s.floor = s.cadence.Complete(s.now())
		} else {
			s.cooldown()
		}
		return nil
	}
	closeErr := s.disconnect()
	if errors.Is(err, screenclient.ErrResync) {
		return errors.Join(translate(err), closeErr)
	}
	if s.client.Pending() {
		return errors.Join(screendelivery.ErrTransportLost, err, closeErr)
	}
	return errors.Join(err, closeErr)
}

func (s *Sender) sendFrame(frame display.Frame, options refreshpolicy.Options) error {
	if s.rebind {
		if err := s.client.Rebind(); err != nil {
			return err
		}
	}
	if s.cadence.Enabled() {
		return s.client.SendWithOptions(frame, options, s.cadence.Policy)
	}
	if options.Priority == refreshpolicy.Urgent || options.Mode == refreshpolicy.Partial {
		return screenclient.ErrUnsupportedRefresh
	}
	return s.client.Send(frame)
}

// ResetSession is explicit permission to abandon unproven state, not permission
// to resend. The owner must invalidate its baseline before calling this method.
func (s *Sender) ResetSession() {
	// Retain an unreaped worker: process() cannot start a replacement over it.
	_ = s.disconnect()
	s.client = newClient()
	s.floor = time.Time{}
	s.cadence.Reset()
}

func (s *Sender) cooldown() {
	if s.cadence.Enabled() {
		s.floor = s.cadence.Unknown(s.now(), s.floor, time.Duration(s.caps.MinimumFullMS)*time.Millisecond)
		return
	}
	next := s.now().Add(time.Duration(s.caps.MinimumFullMS) * time.Millisecond)
	if next.After(s.floor) {
		s.floor = next
	}
	s.cadence.Urgent = s.floor
}

func (s *Sender) ConfigureRefresh(policy refreshpolicy.Policy) error {
	return s.cadence.Configure(policy)
}

func operation(ctx context.Context, p worker, fn func() error) error {
	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()
	finished := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { _ = p.Close(); close(finished) })
	err := fn()
	if !stop() {
		<-finished
	}
	return errors.Join(err, ctx.Err())
}

func translate(err error) error {
	if errors.Is(err, screenclient.ErrResync) {
		return errors.Join(screendelivery.ErrTransportResync, err)
	}
	return err
}

func retryable(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, os.ErrClosed) || errors.Is(err, syscall.EPIPE)
}
