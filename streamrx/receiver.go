package streamrx

import (
	"errors"
	"time"
)

// Begin never repeats a confirmed refresh. A failed intent needs a new ID and
// full upload; a new receiver/epoch cannot attest a previous epoch's outcome.
func (r *Receiver) Begin(intent Intent, now time.Duration) error {
	if err := r.Tick(now); err != nil {
		return err
	}
	if r.status.State == Complete && intent == r.status.Intent {
		return nil
	}
	if r.active() || r.status.State == Closed {
		return ErrState
	}
	if intent.Epoch != r.config.Epoch || intent.ID == 0 || intent.ID <= r.status.Intent.ID {
		return ErrState
	}
	r.status = Status{Intent: intent, State: Receiving}
	r.started, r.progress = now, now
	r.digest.Reset()
	if err := r.sink.Begin(); err != nil {
		return r.fail(err)
	}
	return nil
}

// Write accepts exactly the next nonempty chunk. Duplicates are rejected rather
// than replayed into an unknown controller write pointer. Authentication must
// precede this call; SHA-256 consistency is not sender authentication.
func (r *Receiver) Write(intent Intent, pass uint8, offset uint32, pixels []byte, now time.Duration) error {
	if err := r.Tick(now); err != nil {
		return err
	}
	if r.status.State != Receiving || intent != r.status.Intent {
		return r.fail(ErrState)
	}
	if pass != r.status.Pass || offset != r.status.Offset || !r.validChunk(offset, pixels) {
		return r.fail(ErrChunk)
	}
	if _, err := r.digest.Write(pixels); err != nil {
		return r.fail(err)
	}
	if err := r.sink.Write(pass, offset, pixels); err != nil {
		return r.fail(err)
	}
	r.status.Offset += uint32(len(pixels))
	r.progress = now
	return r.finishPass(intent)
}

func (r *Receiver) finishPass(intent Intent) error {
	if r.status.Offset != r.bytes {
		return nil
	}
	r.digest.Sum(r.sum[:0])
	if r.sum != intent.Digest {
		return r.fail(ErrDigest)
	}
	r.status.Pass++
	r.status.Offset = 0
	r.digest.Reset()
	if r.status.Pass == r.config.Passes {
		r.status.State = Ready
	}
	return nil
}

func (r *Receiver) validChunk(offset uint32, pixels []byte) bool {
	if len(pixels) == 0 || len(pixels) > r.config.MaxChunk || uint64(offset)+uint64(len(pixels)) > uint64(r.bytes) {
		return false
	}
	if r.config.Width%8 == 0 {
		return true
	}
	mask := byte(0xff >> (r.config.Width % 8))
	for i, value := range pixels {
		if (offset+uint32(i)+1)%r.stride == 0 && value&mask != 0 {
			return false
		}
	}
	return true
}

func (r *Receiver) Commit(intent Intent, now time.Duration) error {
	if err := r.Tick(now); err != nil {
		return err
	}
	if r.status.State == Complete && intent == r.status.Intent {
		return nil
	}
	if r.status.State != Ready || intent != r.status.Intent {
		return r.fail(ErrState)
	}
	if err := r.sink.Commit(); err != nil {
		return r.fail(err)
	}
	r.status.State = Complete
	return nil
}

// Tick must run from the owner loop during idle input too. No background timer
// or goroutine is created. Deadlines expire at equality and never extend Total.
func (r *Receiver) Tick(now time.Duration) error {
	if now < 0 || now < r.observed {
		return r.fail(ErrTimeout)
	}
	r.observed = now
	if r.active() && (now-r.started >= r.config.Total || now-r.progress >= r.config.Idle) {
		return r.fail(ErrTimeout)
	}
	return nil
}

func (r *Receiver) Close() error {
	var err error
	if r.active() {
		err = r.sink.Abort()
	}
	r.status.State = Closed
	r.status.Failure = errors.Join(r.status.Failure, err)
	return err
}

func (r *Receiver) active() bool { return r.status.State == Receiving || r.status.State == Ready }

// Invalidate discards active staging after a transport-level validation failure.
func (r *Receiver) Invalidate(cause error) error {
	if cause == nil {
		cause = ErrState
	}
	return r.fail(cause)
}

func (r *Receiver) fail(cause error) error {
	if r.active() {
		cause = errors.Join(cause, r.sink.Abort())
		r.status.State = Failed
		r.status.Failure = cause
	}
	return cause
}
