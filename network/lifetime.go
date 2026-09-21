package network

import (
	"encoding/binary"
	"sync"
)

// Lifetime is created once after epochstore durably reserves a boot epoch.
// This is uniqueness, NOT entropy: no weak platform RNG enters cryptography.
// Pico supplies an authenticated epoch/session challenge; the manager supplies
// a cryptographic random client nonce. Existing HMAC derives session keys.
// See RFC5116 sections3.1–3.2 and SPEC-screen-session.
// https://www.rfc-editor.org/rfc/rfc5116.html#section-3.1
type Lifetime struct {
	mu             sync.Mutex
	boot           [16]byte
	epoch, counter uint64
}

func NewLifetime(uid [8]byte, reservedEpoch uint64) (*Lifetime, error) {
	if uid == [8]byte{} || reservedEpoch == 0 {
		return nil, ErrEntropy
	}
	l := &Lifetime{epoch: reservedEpoch}
	copy(l.boot[:8], uid[:])
	binary.BigEndian.PutUint64(l.boot[8:], reservedEpoch)
	return l, nil
}

// BootID is public board UID + durable epoch. It distinguishes ordinary reboots
// and boards, not authorized journal resets; recovery must reset all host caches.
func (l *Lifetime) BootID() [16]byte { return l.boot }

// NextSession consumes before any handshake/preface bytes can be emitted or
// lost. Keep one Lifetime across reconnects and credential rotation. TCP dialer
// Pico still uses the cryptographic Device role (ServerHandshake).
func (l *Lifetime) NextSession() (epoch, session uint64, err error) {
	if l == nil {
		return 0, 0, ErrEntropy
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.counter == ^uint64(0) {
		return 0, 0, ErrEntropy
	}
	l.counter++ // Consume before a handshake attempt can emit or lose any bytes.
	return l.epoch, l.counter, nil
}
