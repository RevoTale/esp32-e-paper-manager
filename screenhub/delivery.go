//go:build !tinygo

package screenhub

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

var _ screendelivery.RecoveringSender = (*Hub)(nil)

func (h *Hub) WaitReady(ctx context.Context) (screendelivery.Readiness, error) {
	for {
		if err := ctx.Err(); err != nil {
			return screendelivery.Readiness{}, err
		}
		p, err := h.take()
		if err != nil {
			return screendelivery.Readiness{}, err
		}
		if p != nil {
			ready, err := h.connect(ctx, p)
			if err == nil {
				return ready, nil
			}
			if !transient(err) {
				return screendelivery.Readiness{}, translate(err)
			}
		}
		if h.current != nil {
			return h.cadence.Ready(h.floor), nil
		}
		if err := h.wait(ctx); err != nil {
			return screendelivery.Readiness{}, err
		}
	}
}

func (h *Hub) wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-h.done:
		return net.ErrClosed
	case <-h.changed:
		return nil
	}
}

func (h *Hub) matches(caps screenwire.Capabilities) bool {
	return caps.DeviceID == h.config.ID && int(caps.Width) == h.config.Size.Width &&
		int(caps.Height) == h.config.Size.Height && caps.Profile == h.config.Profile && caps.ProfileVersion == h.config.ProfileVersion
}

func (h *Hub) connect(ctx context.Context, p *peer) (screendelivery.Readiness, error) {
	h.disconnect()
	h.current = p
	stop := context.AfterFunc(ctx, func() { _ = p.socket.Close() })
	defer stop()
	caps, err := h.client.Connect(p.stream)
	if err != nil {
		h.disconnect()
		return screendelivery.Readiness{}, err
	}
	if !h.matches(caps) {
		h.disconnect()
		return screendelivery.Readiness{}, ErrProfile
	}
	if h.cadence.Enabled() && caps.Features&screenwire.FeatureRefreshPolicy == 0 {
		h.disconnect()
		return screendelivery.Readiness{}, screenclient.ErrUnsupportedRefresh
	}
	h.caps = caps
	if h.floor.IsZero() {
		h.cooldown()
	}
	ready := h.cadence.Ready(h.floor)
	if !h.client.Pending() {
		return ready, nil
	}
	result, err := h.client.Reconcile()
	if err != nil {
		h.disconnect()
		return screendelivery.Readiness{}, err
	}
	ready.Pending = screendelivery.PendingUnconfirmed
	if result.Confirmed {
		ready.Pending = screendelivery.PendingConfirmed
	}
	h.cooldown() // Lost ACK does not prove when the physical refresh finished.
	ready.NotBefore = h.floor
	ready.UrgentNotBefore = h.floor
	return ready, nil
}

func (h *Hub) Send(ctx context.Context, frame display.Frame) error {
	return h.SendWithOptions(ctx, frame, refreshpolicy.Options{})
}

func (h *Hub) SendWithOptions(ctx context.Context, frame display.Frame, options refreshpolicy.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if h.current == nil {
		return screendelivery.ErrTransportLost
	}
	if h.now().Before(h.cadence.Deadline(h.floor, options)) {
		return ErrCooldown
	}
	p := h.current
	stop := context.AfterFunc(ctx, func() { _ = p.socket.Close() })
	defer stop()
	var err error
	if h.cadence.Enabled() {
		err = h.client.SendWithOptions(frame, options, h.cadence.Policy)
	} else if options.Priority == refreshpolicy.Urgent || options.Mode == refreshpolicy.Partial {
		return screenclient.ErrUnsupportedRefresh
	} else {
		err = h.client.Send(frame)
	}
	if err == nil {
		if h.cadence.Enabled() {
			h.floor = h.cadence.Complete(h.now())
		} else {
			h.cooldown()
		}
		return nil
	}
	h.disconnect()
	if transient(err) {
		return errors.Join(screendelivery.ErrTransportLost, err)
	}
	return translate(err)
}

// ResetSession requires the caller to have invalidated its pixel baseline and
// resolved any retained cycle first. It cannot itself resend or confirm content.
func (h *Hub) ResetSession() {
	h.disconnect()
	h.client = newClient()
	h.floor = time.Time{}
	h.cadence.Urgent = time.Time{}
}

func (h *Hub) cooldown() {
	if h.cadence.Enabled() {
		h.floor = h.cadence.Unknown(h.now(), h.floor, time.Duration(h.caps.MinimumFullMS)*time.Millisecond)
		return
	}
	next := h.now().Add(time.Duration(h.caps.MinimumFullMS) * time.Millisecond)
	if next.After(h.floor) {
		h.floor = next
	}
	h.cadence.Urgent = h.floor
}

func (h *Hub) ConfigureRefresh(policy refreshpolicy.Policy) error { return h.cadence.Configure(policy) }

func (h *Hub) disconnect() {
	if h.current != nil {
		h.drop(h.current.socket)
		h.current = nil
	}
}

func translate(err error) error {
	if errors.Is(err, screenclient.ErrResync) {
		return errors.Join(screendelivery.ErrTransportResync, err)
	}
	return err
}

func transient(err error) bool {
	// A joined cleanup failure cannot turn invalid protocol/authentication into
	// a retryable socket error, nor hide a required lease resynchronization.
	for _, fatal := range []error{screenwire.ErrRecord, screenclient.ErrResync, screenclient.ErrPending,
		securetransport.ErrAuthentication, securetransport.ErrBounds, securetransport.ErrConfig, securetransport.ErrSequence} {
		if errors.Is(err, fatal) {
			return false
		}
	}
	var remote screenclient.RemoteError
	if errors.As(err, &remote) {
		return false
	}
	var networkError net.Error
	return errors.As(err, &networkError) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, io.ErrClosedPipe) || errors.Is(err, net.ErrClosed)
}
