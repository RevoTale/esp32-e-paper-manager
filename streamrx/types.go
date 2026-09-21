// Package streamrx validates bounded full-image passes before allowing refresh.
// It is an experimental single-owner state machine, not a wire codec or auth
// layer. Callers must authenticate chunks, serialize access, poll Tick even when
// input stops, and Close on disconnect. No controller commands live here.
package streamrx

import (
	"crypto/sha256"
	"errors"
	"hash"
	"time"
)

var (
	ErrConfig  = errors.New("stream: configuration")
	ErrState   = errors.New("stream: state or intent conflict")
	ErrChunk   = errors.New("stream: chunk bounds, order or padding")
	ErrDigest  = errors.New("stream: pass digest mismatch")
	ErrTimeout = errors.New("stream: deadline or clock regression")
)

type State uint8

const (
	Idle State = iota
	Receiving
	Ready
	Complete
	Failed
	Closed
)

// Config is negotiated locally; dimensions and passes cannot change mid-session.
// Epoch must be fresh on reboot/reconnect; its generation belongs to transport.
// Idle/Total bound input waiting, not synchronous sink execution or panel BUSY.
type Config struct {
	Epoch         uint64
	Width, Height uint16
	Passes        uint8
	MaxChunk      int
	Idle, Total   time.Duration
}

// Intent identifies one immutable target. ID increases strictly within Epoch.
// Digest covers canonical 1bpp pixels, 1=black, MSB-first, zero row padding.
// Every pass carries identical logical pixels; the sink owns plane polarity.
type Intent struct {
	Epoch, ID uint64
	Digest    [sha256.Size]byte
}

// Sink borrows Write bytes only until return. Begin prepares staging without
// refresh; Commit owns safe refresh cadence/BUSY/sleep. Abort must release power
// without refreshing, including after a partially failed Begin or Commit.
// Sink calls must be bounded; reentry into Receiver is forbidden.
type Sink interface {
	Begin() error
	Write(pass uint8, offset uint32, pixels []byte) error
	Commit() error
	Abort() error
}

type Status struct {
	Intent  Intent
	State   State
	Pass    uint8
	Offset  uint32
	Failure error
}

type Receiver struct {
	config                      Config
	sink                        Sink
	digest                      hash.Hash
	sum                         [sha256.Size]byte
	status                      Status
	bytes, stride               uint32
	started, progress, observed time.Duration
}

func New(config Config, sink Sink) (*Receiver, error) {
	if sink == nil || !config.valid() {
		return nil, ErrConfig
	}
	stride := (uint32(config.Width) + 7) / 8
	return &Receiver{config: config, sink: sink, digest: sha256.New(),
		stride: stride, bytes: stride * uint32(config.Height)}, nil
}

func (c Config) valid() bool {
	return c.Epoch != 0 && c.Width != 0 && c.Height != 0 && c.Passes >= 1 &&
		c.Passes <= 2 && c.MaxChunk >= 1 && c.MaxChunk <= 4096 && c.Idle > 0 && c.Total >= c.Idle
}

func (r *Receiver) Status() Status { return r.status }
