package manager

import (
	"errors"
	"sync"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
	"github.com/RevoTale/esp32-e-paper-manager/update"
)

var (
	ErrConflict = errors.New("manager: idempotency conflict")
	ErrNoUpdate = errors.New("manager: no pending update")
)

type Pending struct {
	Request   update.Request
	ExpiresAt time.Time
	Accepted  bool
}

type DeviceStatus struct {
	Pending    bool
	Connected  bool
	LastSeen   time.Time
	LastResult update.Result
	Generation uint64
}

type deviceState struct {
	pending    *Pending
	connected  bool
	lastSeen   time.Time
	result     update.Result
	generation uint64
}

type Store struct {
	mu      sync.Mutex
	devices map[securetransport.DeviceID]*deviceState
	path    string
}

func NewStore() *Store {
	return &Store{devices: make(map[securetransport.DeviceID]*deviceState)}
}

func (s *Store) Submit(id securetransport.DeviceID, request update.Request, now time.Time,
	ttl time.Duration,
) (uint64, bool, error) {
	if s == nil || id == (securetransport.DeviceID{}) || request.Validate() != nil || ttl <= 0 {
		return 0, false, ErrConfiguration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.state(id)
	before := *state
	s.expire(state, now)
	if state.pending != nil && state.pending.Request.ID() == request.ID() {
		if state.pending.Request.ContentHash() != request.ContentHash() {
			return state.generation, false, ErrConflict
		}
		return state.generation, false, nil
	}
	state.generation++
	state.pending = &Pending{Request: request, ExpiresAt: now.Add(ttl)}
	if err := s.persistLocked(); err != nil {
		*state = before
		return 0, false, err
	}
	return state.generation, true, nil
}

func (s *Store) Lease(id securetransport.DeviceID, now time.Time) (Pending, uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.state(id)
	s.expire(state, now)
	state.connected, state.lastSeen = true, now
	if state.pending == nil || state.pending.Accepted {
		return Pending{}, state.generation, ErrNoUpdate
	}
	return *state.pending, state.generation, nil
}

func (s *Store) Complete(id securetransport.DeviceID, generation uint64, result update.Result,
	now time.Time,
) error {
	if result.Validate() != nil {
		return ErrConfiguration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.state(id)
	before := *state
	if state.pending == nil {
		if state.generation == generation && state.result == result {
			return nil
		}
		return ErrConflict
	}
	if state.generation != generation || state.pending.Request.ID() != result.ID {
		return ErrConflict
	}
	state.result = result
	if result.Status == update.StatusAccepted {
		state.pending.Accepted = true
	} else {
		state.pending = nil
	}
	state.connected, state.lastSeen = true, now
	if err := s.persistLocked(); err != nil {
		*state = before
		return err
	}
	return nil
}

// ReopenAccepted makes an acknowledged update deliverable after a device
// reconnects without reporting any retained runtime state. That indicates a
// reboot before the physical refresh completed.
func (s *Store) ReopenAccepted(id securetransport.DeviceID, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.state(id)
	s.expire(state, now)
	if state.pending == nil || !state.pending.Accepted {
		return nil
	}
	state.pending.Accepted = false
	if err := s.persistLocked(); err != nil {
		state.pending.Accepted = true
		return err
	}
	return nil
}

func (s *Store) Disconnect(id securetransport.DeviceID, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.state(id)
	state.connected, state.lastSeen = false, now
}

func (s *Store) Status(id securetransport.DeviceID, now time.Time) DeviceStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.state(id)
	s.expire(state, now)
	return DeviceStatus{Pending: state.pending != nil, Connected: state.connected,
		LastSeen: state.lastSeen, LastResult: state.result, Generation: state.generation}
}

func (s *Store) state(id securetransport.DeviceID) *deviceState {
	state := s.devices[id]
	if state == nil {
		state = &deviceState{}
		s.devices[id] = state
	}
	return state
}

func (s *Store) expire(state *deviceState, now time.Time) {
	if state.pending != nil && !now.Before(state.pending.ExpiresAt) {
		state.pending = nil
	}
}
