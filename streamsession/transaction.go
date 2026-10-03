package streamsession

import (
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func (s *Session) check(binding Binding, tx Transaction) error {
	if !s.valid(binding) || tx.Lease != s.lease || tx.ID == 0 {
		return ErrLease
	}
	return nil
}

func (s *Session) Begin(binding Binding, tx Transaction, now time.Duration) error {
	if err := s.check(binding, tx); err != nil {
		return err
	}
	if err := s.Tick(now); err != nil {
		return err
	}
	if s.active() {
		return ErrBusy
	}
	if tx.ID <= s.consumed {
		return ErrStale
	}
	if s.Cooldown() != 0 {
		return ErrCooldown
	}
	rx, err := streamrx.New(s.config, s.sink)
	if err != nil {
		return err
	}
	s.rx, s.consumed = rx, tx.ID
	err = s.rx.Begin(tx.intent(), now)
	s.capture()
	return err
}

func (s *Session) Write(binding Binding, tx Transaction, pass uint8, offset uint32, pixels []byte, now time.Duration) error {
	if err := s.check(binding, tx); err != nil {
		return err
	}
	if err := s.Tick(now); err != nil {
		return err
	}
	if s.rx == nil {
		return ErrStale
	}
	err := s.rx.Write(tx.intent(), pass, offset, pixels, now)
	s.capture()
	return err
}

func (s *Session) Commit(binding Binding, tx Transaction, now time.Duration) error {
	if err := s.check(binding, tx); err != nil {
		return err
	}
	if err := s.Tick(now); err != nil {
		return err
	}
	result, err := s.Query(binding, tx)
	if err != nil {
		return err
	}
	if result.Status.State == streamrx.Complete && result.CurrentImage {
		if s.active() {
			return ErrBusy
		}
		return nil
	}
	if result.Status.State != streamrx.Ready {
		return streamrx.ErrState
	}
	// The panel may change before a late BUSY/power-off error. Invalidate old
	// equivalence before calling hardware, and retain failure before replying.
	s.proof, s.lastRefresh = false, now
	err = s.rx.Commit(tx.intent(), now)
	if err == nil {
		s.proof = true
	}
	s.capture()
	return err
}

// Query distinguishes an unseen future ID from consumed IDs whose evidence has
// been evicted. A failed/aborted ID can never restart a controller RAM pointer.
func (s *Session) Query(binding Binding, tx Transaction) (Result, error) {
	if err := s.check(binding, tx); err != nil {
		return Result{}, err
	}
	if s.rx != nil && s.rx.Status().Intent.ID == tx.ID {
		return s.result(s.rx.Status(), tx)
	}
	if s.terminal.Intent.ID == tx.ID {
		return s.result(s.terminal, tx)
	}
	if tx.ID <= s.consumed {
		return Result{}, ErrStale
	}
	return Result{Status: streamrx.Status{Intent: tx.intent(), State: streamrx.Idle}}, nil
}

func (s *Session) result(status streamrx.Status, tx Transaction) (Result, error) {
	if status.Intent.Digest != tx.Digest {
		return Result{}, ErrConflict
	}
	return Result{Status: status, CurrentImage: s.proof && status.State == streamrx.Complete}, nil
}

func (s *Session) capture() {
	if s.rx == nil || s.active() {
		return
	}
	s.terminal = s.rx.Status()
}

// Disconnect must use the binding of the connection that actually closed.
// Completed evidence survives; partial staging closes without a refresh.
func (s *Session) Disconnect(binding Binding) error {
	if !s.valid(binding) {
		return ErrLease
	}
	var err error
	if s.active() {
		err = s.rx.Close()
		s.capture()
	}
	s.rx = nil
	s.writer = Binding{}
	return err
}

// Tick runs even when no input arrives. Regressing clocks invalidate staging;
// observations never go backwards across receiver replacements or leases.
func (s *Session) Tick(now time.Duration) error {
	if now < 0 || now < s.observed {
		return s.invalidate(streamrx.ErrTimeout)
	}
	s.observed = now
	if s.rx == nil {
		return nil
	}
	err := s.rx.Tick(now)
	s.capture()
	return err
}

func (s *Session) invalidate(cause error) error {
	if s.active() {
		cause = s.rx.Invalidate(cause)
		s.capture()
	}
	return cause
}

// Abort is also used for authenticated framing failures and USB preemption.
// It never releases the binding: only its owner may disconnect it.
func (s *Session) Abort(binding Binding) error {
	if !s.valid(binding) {
		return ErrLease
	}
	if !s.active() {
		return nil
	}
	err := s.rx.Close()
	s.capture()
	return err
}
