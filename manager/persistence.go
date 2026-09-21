package manager

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
	"github.com/RevoTale/esp32-e-paper-manager/update"
)

const maxStateBytes = 128 << 10

type diskState struct {
	Devices []diskDevice `json:"devices"`
}

type diskDevice struct {
	ID         securetransport.DeviceID `json:"id"`
	Pending    []byte                   `json:"pending,omitempty"`
	ExpiresAt  time.Time                `json:"expires_at,omitempty"`
	Result     []byte                   `json:"result,omitempty"`
	Generation uint64                   `json:"generation"`
	Accepted   bool                     `json:"accepted,omitempty"`
}

func NewPersistentStore(path string) (*Store, error) {
	if path == "" {
		return nil, ErrConfiguration
	}
	store := NewStore()
	store.path = path
	if err := store.load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return store, nil
}

func (s *Store) load() error {
	info, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0o077 != 0 || info.Size() > maxStateBytes {
		return ErrConfiguration
	}
	file, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	decoder := json.NewDecoder(io.LimitReader(file, maxStateBytes+1))
	decoder.DisallowUnknownFields()
	var state diskState
	if decoder.Decode(&state) != nil || len(state.Devices) > 128 {
		return ErrConfiguration
	}
	for _, device := range state.Devices {
		if err := s.restore(device); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) restore(device diskDevice) error {
	if device.ID == (securetransport.DeviceID{}) || device.Generation == 0 {
		return ErrConfiguration
	}
	state := &deviceState{generation: device.Generation}
	if len(device.Pending) > 0 {
		request, err := update.Decode(device.Pending)
		if err != nil || device.ExpiresAt.IsZero() {
			return ErrConfiguration
		}
		state.pending = &Pending{Request: request, ExpiresAt: device.ExpiresAt, Accepted: device.Accepted}
	} else if device.Accepted {
		return ErrConfiguration
	}
	if len(device.Result) > 0 {
		result, err := update.DecodeResult(device.Result)
		if err != nil {
			return ErrConfiguration
		}
		state.result = result
	}
	if s.devices[device.ID] != nil {
		return ErrConfiguration
	}
	s.devices[device.ID] = state
	return nil
}

func (s *Store) persistLocked() error {
	if s.path == "" {
		return nil
	}
	state, err := s.diskState()
	if err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil || len(data) > maxStateBytes {
		return ErrConfiguration
	}
	file, err := os.CreateTemp(filepath.Dir(s.path), ".epaper-state-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer func() { _ = os.Remove(temporary) }()
	if err = file.Chmod(0o600); err == nil {
		_, err = file.Write(append(data, '\n'))
	}
	if syncErr := file.Sync(); err == nil {
		err = syncErr
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temporary, s.path)
}

func (s *Store) diskState() (diskState, error) {
	state := diskState{Devices: make([]diskDevice, 0, len(s.devices))}
	for id, current := range s.devices {
		if current.generation == 0 {
			continue
		}
		device := diskDevice{ID: id, Generation: current.generation}
		if current.pending != nil {
			device.Pending = make([]byte, update.EncodedSize(current.pending.Request))
			if _, err := update.Encode(device.Pending, current.pending.Request); err != nil {
				return diskState{}, err
			}
			device.ExpiresAt = current.pending.ExpiresAt
			device.Accepted = current.pending.Accepted
		}
		if current.result.ID != (update.ID{}) {
			device.Result = make([]byte, update.ResultEncodedSize)
			if _, err := update.EncodeResult(device.Result, current.result); err != nil {
				return diskState{}, err
			}
		}
		state.Devices = append(state.Devices, device)
	}
	return state, nil
}
