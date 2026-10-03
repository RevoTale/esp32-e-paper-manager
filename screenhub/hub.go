//go:build !tinygo

// Package screenhub accepts bounded EPN2 connections for one enrolled display.
// One ScreenPump owns all EPS2 client state; network workers only authenticate
// and offer peers. Reconciliation never repeats a physical refresh.
package screenhub

import (
	"crypto/rand"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenpeer"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

var (
	ErrCapacity = errors.New("screen handshake capacity reached")
	ErrProfile  = errors.New("screen identity or physical profile differs from configuration")
	ErrCooldown = errors.New("screen physical refresh cooldown has not elapsed")
)

type Config struct {
	ID             securetransport.DeviceID
	Key            securetransport.Key
	Size           display.Size
	Profile        uint32
	ProfileVersion uint16
}

type peer struct {
	socket screenpeer.Socket
	stream *screenpeer.HostScreen
}

type Hub struct {
	config  Config
	mu      sync.Mutex
	sockets map[screenpeer.Socket]struct{}
	closed  bool
	done    chan struct{}
	slots   chan struct{}
	changed chan struct{}
	queued  *peer
	// Fields below belong exclusively to the caller of WaitReady/Send/ResetSession.
	current *peer
	client  *screenclient.Client
	caps    screenwire.Capabilities
	floor   time.Time
	cadence screendelivery.Cadence
	now     func() time.Time
}

func New(config Config) (*Hub, error) {
	if config.ID == (securetransport.DeviceID{}) || config.Key == (securetransport.Key{}) ||
		config.Size.Width <= 0 || config.Size.Width > 65535 || config.Size.Height <= 0 || config.Size.Height > 65535 ||
		config.Profile == 0 || config.ProfileVersion == 0 {
		return nil, securetransport.ErrConfig
	}
	return &Hub{config: config, sockets: make(map[screenpeer.Socket]struct{}),
		done: make(chan struct{}), slots: make(chan struct{}, 4), changed: make(chan struct{}, 1),
		client: newClient(), now: time.Now}, nil
}

func newClient() *screenclient.Client {
	client, err := screenclient.New(rand.Reader)
	if err != nil {
		panic(err)
	} // Non-nil standard-library randomness satisfies the constructor.
	return client
}

func (h *Hub) Changed() <-chan struct{} { return h.changed }

func (h *Hub) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil
	}
	h.closed = true
	close(h.done)
	// An idle pump listens to Changed even when no author scene exists yet.
	select {
	case h.changed <- struct{}{}:
	default:
	}
	var result error
	for socket := range h.sockets {
		result = errors.Join(result, socket.Close())
	}
	clear(h.sockets)
	h.queued = nil
	return result
}

func (h *Hub) drop(socket screenpeer.Socket) {
	h.mu.Lock()
	delete(h.sockets, socket)
	h.mu.Unlock()
	// Close may already have run during context cancellation or failed AEAD I/O.
	_ = socket.Close()
}

func (h *Hub) reserve(socket screenpeer.Socket) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		_ = socket.Close()
		return net.ErrClosed
	}
	select {
	case h.slots <- struct{}{}:
		h.sockets[socket] = struct{}{}
		return nil
	default:
		_ = socket.Close()
		return ErrCapacity
	}
}
