package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

func TestRegistryLoadAndLookup(t *testing.T) {
	record := DeviceRecord{ID: securetransport.DeviceID{1}, Key: securetransport.Key{2}, Timezone: "Europe/Kyiv"}
	registry, err := NewStaticRegistry([]DeviceRecord{record})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := registry.Lookup(record.ID); !ok || got != record {
		t.Fatalf("record=%+v ok=%v", got, ok)
	}
	if _, ok := registry.Lookup(securetransport.DeviceID{9}); ok {
		t.Fatal("unknown device found")
	}
	if _, err = NewStaticRegistry([]DeviceRecord{record, record}); err == nil {
		t.Fatal("duplicate accepted")
	}
}

func TestRegistryRejectsInvalidRecordsAndNilLookup(t *testing.T) {
	for _, record := range []DeviceRecord{
		{},
		{ID: securetransport.DeviceID{1}, Key: securetransport.Key{2}, Timezone: "Invalid/Zone"},
	} {
		if _, err := NewStaticRegistry([]DeviceRecord{record}); err == nil {
			t.Fatalf("accepted %+v", record)
		}
	}
	var registry *StaticRegistry
	if _, ok := registry.Lookup(securetransport.DeviceID{1}); ok {
		t.Fatal("nil registry found record")
	}
}

func TestLoadEnrollmentRequiresPrivateValidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "device.json")
	enrollment := hostprovision.Enrollment{DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Timezone: "Europe/Kyiv"}
	data, _ := json.Marshal(enrollment)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadEnrollment(path)
	if err != nil || loaded.ID[0] != 1 || loaded.Key[0] != 2 {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	if err = os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadEnrollment(path); err == nil {
		t.Fatal("public enrollment accepted")
	}
}

func TestLoadEnrollmentRejectsMalformedAndInvalidRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "device.json")
	for _, data := range [][]byte{
		[]byte(`{"bad":`),
		[]byte(`{"device_id":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"device_key":[2,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"timezone":"Europe/Kyiv"}`),
	} {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadEnrollment(path); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}
