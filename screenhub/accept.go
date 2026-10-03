//go:build !tinygo

package screenhub

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenpeer"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

// Serve owns the listener. It creates at most four concurrent handshake workers;
// closing its context closes every socket and joins those workers.
func (h *Hub) Serve(ctx context.Context, listener net.Listener) error {
	if listener == nil {
		return securetransport.ErrConfig
	}
	stop := context.AfterFunc(ctx, func() { _ = listener.Close(); _ = h.Close() })
	defer stop()
	var workers sync.WaitGroup
	defer workers.Wait()
	defer func() { _ = listener.Close(); _ = h.Close() }()
	for {
		socket, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if err = h.reserve(socket); err != nil {
			if errors.Is(err, net.ErrClosed) {
				// Cancellation can close the hub after Accept, before admission.
				// Preserve the same cause as cancellation while Accept is blocked.
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return err
			}
			continue
		}
		workers.Add(1)
		go func() { defer workers.Done(); _ = h.authenticate(ctx, socket) }()
	}
}

// Offer authenticates synchronously, for transports that already own acceptance.
// On failure the socket is closed. Success transfers ownership to the hub.
func (h *Hub) Offer(ctx context.Context, socket screenpeer.Socket) error {
	if socket == nil {
		return securetransport.ErrConfig
	}
	if err := h.reserve(socket); err != nil {
		return err
	}
	return h.authenticate(ctx, socket)
}

func (h *Hub) authenticate(ctx context.Context, socket screenpeer.Socket) error {
	defer func() { <-h.slots }()
	stop := context.AfterFunc(ctx, func() { _ = socket.Close() })
	defer stop()
	records, _, err := screenpeer.Manager(socket, h.lookup, time.Now())
	if err != nil {
		h.drop(socket)
		return err
	}
	stream, err := screenpeer.NewHostScreen(socket, records)
	if err != nil {
		h.drop(socket)
		return err
	}
	if ctx.Err() != nil {
		h.drop(socket)
		return ctx.Err()
	}
	return h.enqueue(&peer{socket: socket, stream: stream})
}

func (h *Hub) lookup(id securetransport.DeviceID) (securetransport.Key, error) {
	if id != h.config.ID {
		return securetransport.Key{}, securetransport.ErrAuthentication
	}
	return h.config.Key, nil
}

func (h *Hub) enqueue(p *peer) error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		h.drop(p.socket)
		return net.ErrClosed
	}
	old := h.queued
	h.queued = p
	select {
	case h.changed <- struct{}{}:
	default:
	}
	h.mu.Unlock()
	if old != nil {
		h.drop(old.socket)
	}
	return nil
}

func (h *Hub) take() (*peer, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, net.ErrClosed
	}
	p := h.queued
	h.queued = nil
	return p, nil
}
