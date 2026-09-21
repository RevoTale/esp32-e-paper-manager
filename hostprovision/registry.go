package hostprovision

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

var ErrRegistry = errors.New("host provisioning: registry write failed")

func SaveEnrollment(path string, enrollment Enrollment) error {
	if path == "" || enrollment.DeviceID == ([16]byte{}) || enrollment.DeviceKey == ([32]byte{}) {
		return ErrRegistry
	}
	data, err := json.Marshal(enrollment)
	if err != nil {
		return ErrRegistry
	}
	directory := filepath.Dir(path)
	file, err := os.CreateTemp(directory, ".epaper-enrollment-*")
	if err != nil {
		return ErrRegistry
	}
	temporary := file.Name()
	defer func() { _ = os.Remove(temporary) }()
	err = writeEnrollmentFile(file, append(data, '\n'))
	if err != nil || os.Rename(temporary, path) != nil {
		return ErrRegistry
	}
	return verifyPrivate(path)
}

func writeEnrollmentFile(file *os.File, data []byte) error {
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func verifyPrivate(path string) error {
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != fs.FileMode(0o600) {
		return ErrRegistry
	}
	return nil
}
