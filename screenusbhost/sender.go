// Package screenusbhost isolates EPS2 USB I/O in one bounded child process.
// One caller owns readiness, sending and session reset. Close may be concurrent.
// No display frame or private lease is passed to the child as process arguments.
package screenusbhost

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

var (
	ErrConfiguration = errors.New("screen USB: invalid local configuration or geometry")
	ErrWorker        = errors.New("screen USB: serial proxy failed")
	ErrClosed        = errors.New("screen USB: closed")
	ErrCooldown      = errors.New("screen USB: physical refresh cooldown has not elapsed")
)

type worker interface {
	io.ReadWriteCloser
	Done() <-chan struct{}
}

type Sender struct {
	size    display.Size
	mu      sync.Mutex
	current worker
	closed  bool
	fault   error
	changed chan struct{}
	done    chan struct{}
	watched chan struct{}
	start   func() (worker, error)
	retry   func(context.Context) error
	now     func() time.Time
	// The delivery owner alone accesses these fields.
	client  *screenclient.Client
	caps    screenwire.Capabilities
	floor   time.Time
	cadence screendelivery.Cadence
	bound   bool
	rebind  bool
}

var _ screendelivery.RecoveringSender = (*Sender)(nil)

// New takes trusted local paths and expected logical geometry. No enrollment is
// required for trusted physical USB; capabilities still validate EPS2 semantics.
func New(executable, port string, size display.Size) (*Sender, error) {
	if executable == "" || port == "" || size.Width <= 0 || size.Height <= 0 ||
		size.Width > 2048 || size.Height > 2048 || size.Width*size.Height > 1_048_576 {
		return nil, ErrConfiguration
	}
	s := &Sender{size: size, changed: make(chan struct{}, 1), done: make(chan struct{}), watched: make(chan struct{}), client: newClient(), now: time.Now}
	s.rebind = esp32Endpoint(port)
	s.start = func() (worker, error) { return startProcess(executable, port, s.changed) }
	s.retry = s.waitRetry
	go s.watch()
	return s, nil
}

func newClient() *screenclient.Client {
	// TinyGo 0.42 CDC RX retains 512 bytes, dropping excess packet bytes:
	// https://github.com/tinygo-org/tinygo/blob/v0.42.0/src/machine/usb/cdc/usbcdc.go
	// A complete record must fit before the owner runs. Stop-and-wait ACKs,
	// not sleeps, ensure the prior request was drained before the next burst.
	c, err := screenclient.NewWithMaxChunk(rand.Reader, 512-screenwire.HeaderSize)
	if err != nil {
		panic(err)
	} // Non-nil standard-library randomness and fixed valid limit satisfy New.
	return c
}

func (s *Sender) Changed() <-chan struct{} { return s.changed }

func (s *Sender) Close() error {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		close(s.done)
	}
	p := s.current
	s.mu.Unlock()
	<-s.watched
	s.signal()
	if p != nil {
		return p.Close()
	}
	return nil
}

func (s *Sender) signal() {
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

func (s *Sender) waitRetry(ctx context.Context) error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return ErrClosed
	case <-timer.C:
		return nil
	}
}

func (s *Sender) disconnect() error {
	s.bound = false
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil
	}
	// Do not replace a worker until it is reaped, even after a kill failure.
	if err := s.current.Close(); err != nil {
		s.fault = err
		return err
	}
	s.current = nil
	return nil
}

func (s *Sender) process() (worker, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrClosed
	}
	if s.fault != nil {
		return nil, s.fault
	}
	if s.current != nil {
		return s.current, nil
	}
	p, err := s.start()
	if err != nil {
		return nil, err
	}
	s.current = p
	return p, nil
}
