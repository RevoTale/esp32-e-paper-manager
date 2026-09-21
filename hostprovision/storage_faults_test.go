package hostprovision

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func registryTestOps() registryOps {
	return registryOps{write: writeEnrollmentFile, sync: syncRegistry,
		rename: func(root *os.Root, old, next string) error { return root.Rename(old, next) }}
}

func TestStagingFailurePreservesActiveAndLeavesEvidence(t *testing.T) {
	for _, failure := range []string{"write", "sync", "readback", "changed"} {
		t.Run(failure, func(t *testing.T) {
			_, _, enrollment := candidateFixture()
			path := filepath.Join(t.TempDir(), "device.json")
			if err := SaveEnrollment(path, enrollment); err != nil {
				t.Fatal(err)
			}
			previous := mustRead(t, path)
			ops := failingStageOps(failure)
			if _, err := stageEnrollment(path, enrollment, ops); !errors.Is(err, ErrStaging) {
				t.Fatalf("stage=%v", err)
			}
			if !bytes.Equal(mustRead(t, path), previous) {
				t.Fatal("active changed on staging failure")
			}
			if _, err := os.Lstat(path + ".pending"); err != nil {
				t.Fatal("failure evidence removed", err)
			}
			if _, err := StageEnrollment(path, enrollment); !errors.Is(err, ErrPending) {
				t.Fatalf("retry not blocked: %v", err)
			}
		})
	}
}

func failingStageOps(failure string) registryOps {
	ops := registryTestOps()
	switch failure {
	case "write":
		ops.write = func(file *os.File, data []byte) error {
			_, err := file.Write(data[:12])
			return errors.Join(err, file.Close(), io.ErrShortWrite)
		}
	case "sync":
		ops.sync = func(*os.Root) error { return io.ErrClosedPipe }
	case "readback":
		ops.sync = func(root *os.Root) error { return root.Chmod("device.json.pending", 0o644) }
	case "changed":
		ops.sync = func(root *os.Root) error {
			_, _, enrollment := candidateFixture()
			enrollment.DeviceKey[0]++
			data, err := candidateBytes("device.json", enrollment)
			if err != nil {
				return err
			}
			return root.WriteFile("device.json.pending", data, 0o600)
		}
	}
	return ops
}

func TestPromotionFailurePreservesActiveAndCandidate(t *testing.T) {
	request, response, enrollment := candidateFixture()
	path := filepath.Join(t.TempDir(), "device.json")
	old := enrollment
	old.DeviceKey[0]++
	if err := SaveEnrollment(path, old); err != nil {
		t.Fatal(err)
	}
	previous := mustRead(t, path)
	pending, err := StageEnrollment(path, enrollment)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pending.Close() }()
	candidate := mustRead(t, path+".pending")
	pending.ops.rename = func(*os.Root, string, string) error { return io.ErrClosedPipe }
	if err = pending.Confirm(request, response); !errors.Is(err, ErrPromotion) {
		t.Fatalf("promote=%v", err)
	}
	if !bytes.Equal(mustRead(t, path), previous) || !bytes.Equal(mustRead(t, path+".pending"), candidate) {
		t.Fatal("rename failure changed active or candidate")
	}
}

func TestPostRenameSyncFailureReportsInstalledCandidate(t *testing.T) {
	request, response, enrollment := candidateFixture()
	path := filepath.Join(t.TempDir(), "device.json")
	pending, err := StageEnrollment(path, enrollment)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pending.Close() }()
	candidate := mustRead(t, path+".pending")
	pending.ops.sync = func(*os.Root) error { return io.ErrClosedPipe }
	if err = pending.Confirm(request, response); !errors.Is(err, ErrDurability) {
		t.Fatalf("sync=%v", err)
	}
	if !bytes.Equal(mustRead(t, path), candidate) {
		t.Fatal("installed candidate lost")
	}
}

func TestStageOpenFailureAndClosedDirectory(t *testing.T) {
	_, _, enrollment := candidateFixture()
	path := filepath.Join(t.TempDir(), strings.Repeat("a", 250))
	if _, err := StageEnrollment(path, enrollment); !errors.Is(err, ErrStaging) {
		t.Fatalf("long filename=%v", err)
	}
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = root.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = readEnrollmentSnapshot(root, "file"); !errors.Is(err, ErrRegistry) {
		t.Fatalf("closed root=%v", err)
	}
	if err = syncRegistry(root); err == nil {
		t.Fatal("closed directory synchronized")
	}
}

func TestPrivateFileWriteErrors(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "file")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	if err = writeEnrollmentFile(file, []byte("private")); err == nil {
		t.Fatal("closed file accepted")
	}
	file, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = writeEnrollmentFile(file, []byte("private")); err == nil {
		t.Fatal("read-only file accepted")
	}
}
