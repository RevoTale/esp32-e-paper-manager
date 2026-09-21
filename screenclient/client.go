// Package screenclient sends prepared pixels using EPS2 over USB or an already
// authenticated encrypted stream. Retain one Client across reconnects; never
// log its private acquisition claim. It has no automatic physical-update retry.
package screenclient

import (
	"errors"
	"fmt"
	"io"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamsession"
)

var (
	ErrResync     = errors.New("screen client: boot or ownership changed; full resynchronization required")
	ErrPending    = errors.New("screen client: reconcile pending update before sending")
	ErrConnection = errors.New("screen client: connection not bound")
)

// RemoteError carries only typed numeric diagnostics, never device error text.
type RemoteError struct{ Status screenwire.Status }

func (e RemoteError) Error() string {
	return fmt.Sprintf("screen rejected: operation=%d code=%d state=%d pass=%d offset=%d phase=%d step=%d command=%02x busy_known=%t busy=%t",
		e.Status.Operation, e.Status.Code, e.Status.State, e.Status.Pass, e.Status.Offset, e.Status.Diagnostic.Phase, e.Status.Diagnostic.Step,
		e.Status.Diagnostic.Command, e.Status.Diagnostic.BusyKnown, e.Status.Diagnostic.Busy)
}

type acquisition struct {
	epoch streamsession.Epoch
	claim [16]byte
}
type Client struct {
	random    io.Reader
	stream    io.ReadWriter
	buffer    [screenwire.MaxRecord]byte
	lease     streamsession.Lease
	acquiring acquisition
	caps      screenwire.Capabilities
	maxChunk  uint16
	pending   streamsession.Transaction
	next      uint64
	bound     bool
}

// New requires cryptographic randomness in production (normally crypto/rand.Reader).
func New(random io.Reader) (*Client, error) {
	return NewWithMaxChunk(random, screenwire.MaxPayload)
}

// NewWithMaxChunk additionally bounds decoded data bytes per request, independently
// of the negotiated device capabilities. The limit is fixed across reconnects.
// Account for screenwire.HeaderSize in the transport RX budget. Packed encoding
// never enlarges a request; control requests require 64 bytes including headers.
func NewWithMaxChunk(random io.Reader, maxChunk uint16) (*Client, error) {
	if random == nil || maxChunk == 0 || maxChunk > screenwire.MaxPayload {
		return nil, screenwire.ErrRecord
	}
	return &Client{random: random, maxChunk: maxChunk}, nil
}

func (c *Client) Pending() bool { return c.pending.ID != 0 }

// Connect borrows a deadline-bounded stream. The caller must close the previous
// transport first. A changed boot/lease is explicit ErrResync; use a fresh
// Client only after invalidating the manager's unproven pixel baseline.
func (c *Client) Connect(stream io.ReadWriter) (screenwire.Capabilities, error) {
	c.bound = false
	c.stream = stream
	if stream == nil {
		return screenwire.Capabilities{}, ErrConnection
	}
	hello, err := c.exchange(screenwire.Record{Kind: screenwire.Hello})
	if err != nil {
		return screenwire.Capabilities{}, err
	}
	if err = remoteOK(hello.Status); err != nil {
		return screenwire.Capabilities{}, err
	}
	epoch := streamsession.Epoch{Boot: hello.Status.Boot, Generation: hello.Status.Generation}
	if c.lease.Generation == 0 {
		err = c.acquire(epoch)
	} else if c.lease.Epoch != epoch {
		err = ErrResync
	}
	if err != nil {
		return screenwire.Capabilities{}, err
	}
	if c.caps.Profile != 0 && c.caps != hello.Capabilities {
		return screenwire.Capabilities{}, ErrResync
	}
	c.caps = hello.Capabilities
	if err = c.bind(); err != nil {
		return screenwire.Capabilities{}, err
	}
	c.bound = true
	return c.caps, nil
}

func (c *Client) acquireClaim(epoch streamsession.Epoch) error {
	if c.acquiring.claim == [16]byte{} {
		var claim [16]byte
		if _, err := io.ReadFull(progressReader{c.random}, claim[:]); err != nil {
			return err
		}
		if claim == [16]byte{} || epoch.Generation == ^uint64(0) {
			return screenwire.ErrRecord
		}
		c.acquiring = acquisition{epoch: epoch, claim: claim}
	}
	if c.acquiring.epoch.Boot != epoch.Boot {
		return ErrResync
	}
	return nil
}

func (c *Client) acquire(epoch streamsession.Epoch) error {
	if err := c.acquireClaim(epoch); err != nil {
		return err
	}
	a := c.acquiring
	var payload [32]byte
	copy(payload[:16], a.epoch.Boot[:])
	copy(payload[16:], a.claim[:])
	r, err := c.exchange(screenwire.Record{Kind: screenwire.Acquire, Epoch: a.epoch.Generation, Payload: payload[:]})
	if err != nil {
		return err
	}
	if r.Status.Code == screenwire.CodeLease {
		return errors.Join(ErrResync, RemoteError{Status: r.Status})
	}
	if err = remoteOK(r.Status); err != nil {
		return err
	}
	if r.Status.Boot != a.epoch.Boot || r.Status.Generation != a.epoch.Generation+1 {
		return ErrResync
	}
	c.lease = streamsession.Lease{Epoch: streamsession.Epoch{Boot: r.Status.Boot, Generation: r.Status.Generation}, Claim: a.claim}
	c.acquiring = acquisition{}
	return nil
}

func (c *Client) bind() error {
	var payload [32]byte
	copy(payload[:16], c.lease.Boot[:])
	copy(payload[16:], c.lease.Claim[:])
	r, err := c.exchange(screenwire.Record{Kind: screenwire.Bind, Epoch: c.lease.Generation, Payload: payload[:]})
	if err != nil {
		return err
	}
	return c.accept(r.Status)
}

func (c *Client) accept(s screenwire.Status) error {
	if err := c.boundStatus(s); err != nil {
		return err
	}
	return remoteOK(s)
}

func remoteOK(s screenwire.Status) error {
	if s.Code != screenwire.CodeOK {
		return RemoteError{Status: s}
	}
	return nil
}
