package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
)

func TestEnrollmentRejectsOversizedAmbiguousAndUnknownSchema(t *testing.T) {
	valid, _ := json.Marshal(hostprovision.Enrollment{DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Timezone: "UTC"})
	for _, source := range []string{
		string(valid) + strings.Repeat(" ", 1024),
		strings.Replace(string(valid), `"timezone":"UTC"`, `"timezone":"UTC","timezone":"Europe/Kyiv"`, 1),
		strings.Replace(string(valid), `"timezone"`, `"TIMEZONE"`, 1),
		strings.Replace(string(valid), `"timezone":"UTC"`, `"timezone":"UTC","unexpected":1`, 1),
		`{"device_id":[1],"device_key":[2],"timezone":"UTC"}`,
	} {
		path := filepath.Join(t.TempDir(), "enrollment")
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadEnrollment(path); err == nil {
			t.Fatal("unsafe enrollment accepted")
		}
	}
}

func TestEnrollmentRejectsSymlink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "enrollment")
	if err := hostprovision.SaveEnrollment(path, hostprovision.Enrollment{DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Timezone: "UTC"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(path, path+".link"); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadEnrollment(path + ".link"); err == nil {
		t.Fatal("enrollment symlink accepted")
	}
}
