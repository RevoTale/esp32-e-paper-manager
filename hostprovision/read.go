package hostprovision

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/RevoTale/esp32-e-paper-manager/strictjson"
)

// ReadPrivateFile reads an existing regular mode-0600 file, without following
// a final symlink. Its actual read is bounded even if the file grows after stat.
func ReadPrivateFile(path string, maximum int) ([]byte, error) {
	if maximum < 1 || maximum > 4096 {
		return nil, ErrRegistry
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, ErrRegistry
	}
	defer func() { _ = root.Close() }()
	snapshot, err := readPrivateSnapshot(root, filepath.Base(path), maximum)
	if err != nil || snapshot.info == nil {
		return nil, ErrRegistry
	}
	return snapshot.data, nil
}

func LoadEnrollment(path string) (Enrollment, error) {
	data, err := ReadPrivateFile(path, maxEnrollmentBytes)
	if err != nil {
		return Enrollment{}, ErrRegistry
	}
	defer clear(data)
	return DecodeEnrollment(data)
}

// DecodeEnrollment requires the exact private enrollment schema and rejects
// duplicate fields, abbreviated identity arrays, unknown fields and trailing JSON.
func DecodeEnrollment(data []byte) (Enrollment, error) {
	if len(data) > maxEnrollmentBytes || strictjson.Validate(data) != nil {
		return Enrollment{}, ErrRegistry
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil || len(fields) != 3 {
		return Enrollment{}, ErrRegistry
	}
	if !enrollmentArray(fields["device_id"], 16) || !enrollmentArray(fields["device_key"], 32) || fields["timezone"] == nil {
		return Enrollment{}, ErrRegistry
	}
	var enrollment Enrollment
	if json.Unmarshal(data, &enrollment) != nil || !validEnrollment(enrollment) {
		return Enrollment{}, ErrRegistry
	}
	return enrollment, nil
}

func enrollmentArray(data []byte, size int) bool {
	var values []uint8
	return len(data) > 0 && data[0] == '[' && json.Unmarshal(data, &values) == nil && len(values) == size
}
