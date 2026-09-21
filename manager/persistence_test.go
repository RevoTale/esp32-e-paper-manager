package manager

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

func TestPersistentStoreRejectsDuplicateAndExcessDevices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	device := diskDevice{ID: securetransport.DeviceID{1}, Generation: 1}
	for _, state := range []diskState{
		{Devices: []diskDevice{device, device}},
		{Devices: make([]diskDevice, 129)},
	} {
		data, err := json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err = NewPersistentStore(path); !errors.Is(err, ErrConfiguration) {
			t.Fatalf("invalid device list accepted: %v", err)
		}
	}
	store := NewStore()
	store.devices[securetransport.DeviceID{2}] = &deviceState{}
	state, err := store.diskState()
	if err != nil || len(state.Devices) != 0 {
		t.Fatalf("state=%+v err=%v", state, err)
	}
}
