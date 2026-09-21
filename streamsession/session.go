// Package streamsession retains bounded transaction evidence across transports.
// The authenticated owner serializes calls and polls Tick during idle input.
// No method performs authentication, schedules timers or retains image pixels.
package streamsession

import (
	"crypto/sha256"
	"errors"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

var (
	ErrLease    = errors.New("screen: stale boot, lease or binding")
	ErrBusy     = errors.New("screen: writer or transaction active")
	ErrStale    = errors.New("screen: transaction consumed or evidence evicted")
	ErrConflict = errors.New("screen: transaction digest conflict")
	ErrCooldown = errors.New("screen: full refresh cooldown")
)

type Epoch struct {
	Boot       [16]byte
	Generation uint64
}
type Lease struct {
	Epoch
	Claim [16]byte
}
type Binding struct {
	Lease
	Serial uint64
}
type Transaction struct {
	Lease
	ID     uint64
	Digest [sha256.Size]byte
}
type Result struct {
	Status       streamrx.Status
	CurrentImage bool
}

type Session struct {
	config                         streamrx.Config
	sink                           streamrx.Sink
	rx                             *streamrx.Receiver
	lease                          Lease
	writer                         Binding
	serial, consumed               uint64
	terminal                       streamrx.Status
	proof                          bool
	observed, lastRefresh, minimum time.Duration
}

// New conservatively starts with a full cooldown: reboot cannot prove when the
// preceding firmware last refreshed. Boot must come from the board's RNG.
func New(boot [16]byte, config streamrx.Config, sink streamrx.Sink, minimum time.Duration) (*Session, error) {
	if boot == [16]byte{} || minimum <= 0 {
		return nil, streamrx.ErrConfig
	}
	config.Epoch = 1
	if _, err := streamrx.New(config, sink); err != nil {
		return nil, err
	}
	return &Session{config: config, sink: sink, minimum: minimum, lease: Lease{Epoch: Epoch{Boot: boot}}}, nil
}

// Epoch exposes discovery only, never another writer's claim.
func (s *Session) Epoch() Epoch { return s.lease.Epoch }

// Acquire uses compare-and-swap. The caller generates a fresh random 128-bit
// claim for each logical acquisition and retains that same claim for retries.
// An exact lost-ACK retry has no side effects, including when a writer is bound.
func (s *Session) Acquire(expected Epoch, claim [16]byte) (Lease, error) {
	if claim == [16]byte{} || expected.Boot != s.lease.Boot || expected.Generation == ^uint64(0) {
		return Lease{}, ErrLease
	}
	if s.lease.Generation == expected.Generation+1 && s.lease.Claim == claim {
		return s.lease, nil
	}
	if expected != s.Epoch() {
		return Lease{}, ErrLease
	}
	if s.writer.Serial != 0 || s.active() {
		return Lease{}, ErrBusy
	}
	s.lease = Lease{Epoch: Epoch{Boot: expected.Boot, Generation: expected.Generation + 1}, Claim: claim}
	s.config.Epoch = s.lease.Generation
	s.consumed, s.proof, s.rx = 0, false, nil
	s.terminal = streamrx.Status{}
	return s.lease, nil
}

// Bind resumes a remembered lease, not one guessed from Hello. Each connection
// gets a non-reusable serial so a late disconnect cannot close its replacement.
func (s *Session) Bind(lease Lease) (Binding, error) {
	if lease != s.lease || lease.Generation == 0 || lease.Claim == [16]byte{} || s.serial == ^uint64(0) {
		return Binding{}, ErrLease
	}
	if s.writer.Serial != 0 {
		return Binding{}, ErrBusy
	}
	s.serial++
	s.writer = Binding{Lease: lease, Serial: s.serial}
	return s.writer, nil
}

// Fence revokes completion/ownership for credential changes without resetting
// physical cadence. The owner must first disconnect/abort its live binding.
// The new generation is unclaimed; only a fresh Acquire may establish a writer.
func (s *Session) Fence() error {
	if s.writer.Serial != 0 || s.active() {
		return ErrBusy
	}
	if s.lease.Generation == ^uint64(0) {
		return ErrLease
	}
	s.lease.Generation++
	s.lease.Claim = [16]byte{}
	s.consumed, s.proof, s.rx = 0, false, nil
	s.terminal = streamrx.Status{}
	return nil
}

func (s *Session) valid(binding Binding) bool { return binding.Serial != 0 && binding == s.writer }
func (s *Session) active() bool {
	return s.rx != nil && (s.rx.Status().State == streamrx.Receiving || s.rx.Status().State == streamrx.Ready)
}

func (tx Transaction) intent() streamrx.Intent {
	return streamrx.Intent{Epoch: tx.Generation, ID: tx.ID, Digest: tx.Digest}
}

// Cooldown reports the remaining floor without mutating the monotonic clock.
func (s *Session) Cooldown() time.Duration {
	elapsed := s.observed - s.lastRefresh
	if elapsed >= s.minimum {
		return 0
	}
	return s.minimum - elapsed
}
