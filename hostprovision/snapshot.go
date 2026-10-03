package hostprovision

import (
	"bytes"
	"errors"
	"io"
	"os"
)

const maxEnrollmentBytes = 1024

type enrollmentSnapshot struct {
	info os.FileInfo
	data []byte
}

func readEnrollmentSnapshot(root *os.Root, name string) (enrollmentSnapshot, error) {
	snapshot, err := readPrivateSnapshot(root, name, maxEnrollmentBytes)
	if err == nil && snapshot.info != nil && !validEnrollmentData(snapshot.data) {
		return enrollmentSnapshot{}, ErrRegistry
	}
	return snapshot, err
}

func readPrivateSnapshot(root *os.Root, name string, maximum int) (enrollmentSnapshot, error) {
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return enrollmentSnapshot{}, nil
	}
	if err != nil || !privateFile(info, maximum) {
		return enrollmentSnapshot{}, ErrRegistry
	}
	file, err := root.Open(name)
	if err != nil {
		return enrollmentSnapshot{}, ErrRegistry
	}
	defer func() { _ = file.Close() }()
	return readSnapshotFile(file, info, maximum)
}

func readSnapshotFile(file *os.File, info os.FileInfo, maximum int) (enrollmentSnapshot, error) {
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !privateFile(opened, maximum) {
		return enrollmentSnapshot{}, ErrRegistry
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil || len(data) > maximum || int64(len(data)) != info.Size() {
		return enrollmentSnapshot{}, ErrRegistry
	}
	return enrollmentSnapshot{info: info, data: data}, nil
}

func validEnrollmentData(data []byte) bool {
	_, err := DecodeEnrollment(data)
	return err == nil
}

func privateFile(info os.FileInfo, maximum int) bool {
	return info.Mode().IsRegular() && info.Mode().Perm() == 0o600 &&
		info.Size() > 0 && info.Size() <= int64(maximum)
}

func unchangedEnrollment(root *os.Root, name string, expected enrollmentSnapshot) bool {
	actual, err := readEnrollmentSnapshot(root, name)
	if err != nil {
		return false
	}
	if expected.info == nil || actual.info == nil {
		return expected.info == nil && actual.info == nil
	}
	return os.SameFile(expected.info, actual.info) && bytes.Equal(expected.data, actual.data)
}

func syncRegistry(root *os.Root) error {
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
