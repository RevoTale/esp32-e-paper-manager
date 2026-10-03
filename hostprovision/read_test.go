package hostprovision

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateReadBoundsAndIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	data := bytes.Repeat([]byte{'s'}, 4096)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadPrivateFile(path, 4096)
	if err != nil || !bytes.Equal(loaded, data) {
		t.Fatal("bounded read differs", err)
	}
	for _, maximum := range []int{0, 4095, 4097} {
		if _, err = ReadPrivateFile(path, maximum); !errors.Is(err, ErrRegistry) {
			t.Fatalf("limit=%d: %v", maximum, err)
		}
	}
	for _, missing := range []string{filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "missing", "file")} {
		if _, err = ReadPrivateFile(missing, 4096); !errors.Is(err, ErrRegistry) {
			t.Fatal(err)
		}
	}
}

func TestEnrollmentDecodeRequiresExactSchema(t *testing.T) {
	_, _, enrollment := candidateFixture()
	data, err := json.Marshal(enrollment)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"", "[]", "null", "{}", string(data) + "{}",
		string(data) + strings.Repeat(" ", 1024),
		strings.Replace(string(data), `"timezone"`, `"other"`, 1),
		strings.Replace(string(data), `"timezone":"Europe/Kyiv"`, `"timezone":12`, 1),
		strings.Replace(string(data), `"timezone":"Europe/Kyiv"`, `"timezone":""`, 1),
		`{"device_id":[1],"device_key":[2],"timezone":"UTC"}`,
	} {
		if _, err := DecodeEnrollment([]byte(source)); !errors.Is(err, ErrRegistry) {
			t.Fatal("invalid schema accepted")
		}
	}
	if actual, err := DecodeEnrollment(data); err != nil || actual != enrollment {
		t.Fatal("valid enrollment rejected", err)
	}
}

func TestSnapshotRejectsChangedAndClosedFiles(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "snapshot")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.WriteString("secret"); err != nil {
		t.Fatal(err)
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate(3); err != nil {
		t.Fatal(err)
	}
	if _, err = readSnapshotFile(file, info, 4096); !errors.Is(err, ErrRegistry) {
		t.Fatal("changed size accepted")
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = readSnapshotFile(file, info, 4096); !errors.Is(err, ErrRegistry) {
		t.Fatal("closed file accepted")
	}
}
