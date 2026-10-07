// Package managersetup prepares local manager credentials without touching a device.
package managersetup

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
)

// Config selects an existing USB enrollment and a new private destination.
// UID/GID -1 retain the calling user's ownership; Docker setup uses 65532.
type Config struct {
	Enrollment string
	Directory  string
	UID        int
	GID        int
	random     io.Reader
}

func (c Config) entropy() io.Reader {
	if c.random != nil {
		return c.random
	}
	return rand.Reader
}

var filenames = []string{"device.json", "api-token", "tls.crt", "tls.key"}

// Run imports enrollment unchanged and creates loopback TLS and an API token.
// A complete existing destination is verified, never overwritten or repaired.
// The parent must already exist and must not be writable by group or others.
func Run(c Config) error {
	if c.Directory == "" || c.UID < -1 || c.GID < -1 {
		return errors.New("setup: invalid destination or ownership")
	}
	data, err := hostprovision.ReadPrivateFile(c.Enrollment, 4096)
	if err != nil {
		return errors.New("setup: enrollment must be a private regular USB enrollment file")
	}
	defer clear(data)
	if _, err = hostprovision.DecodeEnrollment(data); err != nil {
		return errors.New("setup: invalid USB enrollment")
	}
	return prepare(c, data)
}

func prepare(c Config, enrollment []byte) error {
	parent, err := os.OpenRoot(filepath.Dir(filepath.Clean(c.Directory)))
	if err != nil {
		return err
	}
	defer func() { _ = parent.Close() }()
	info, err := parent.Stat(".")
	if err != nil || info.Mode().Perm()&0o022 != 0 {
		return errors.New("setup: destination parent must not be group/world writable")
	}
	name := filepath.Base(filepath.Clean(c.Directory))
	if name == "." || name == string(filepath.Separator) {
		return errors.New("setup: destination must be a child directory")
	}
	if _, err = parent.Lstat(name); err == nil {
		return validateExisting(parent, name, c, enrollment)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return create(parent, name, c, enrollment)
}

func create(parent *os.Root, name string, c Config, enrollment []byte) error {
	files, err := credentialsWithRandom(enrollment, c.entropy())
	if err != nil {
		return err
	}
	defer func() {
		for _, data := range files {
			clear(data)
		}
	}()
	stage := ".setup-" + rand.Text()
	if err = parent.Mkdir(stage, 0o700); err != nil {
		return err
	}
	// Only remove the exact generated files and directory, never an input path.
	defer cleanupStage(parent, stage)
	for _, file := range filenames {
		if err = writePrivate(parent, filepath.Join(stage, file), files[file], c); err != nil {
			return err
		}
	}
	if err = prepareProducer(parent, stage, c); err != nil {
		return err
	}
	if err = parent.Chown(stage, c.UID, c.GID); err != nil {
		return err
	}
	if err = syncDirectory(parent, stage); err != nil {
		return err
	}
	// A complete nonempty destination cannot be replaced by a racing setup.
	if err = parent.Rename(stage, name); err != nil {
		return err
	}
	return syncDirectory(parent, ".")
}

func cleanupStage(parent *os.Root, stage string) {
	_ = parent.Remove(filepath.Join(stage, "api", "api-token"))
	_ = parent.Remove(filepath.Join(stage, "api", "tls.crt"))
	_ = parent.Remove(filepath.Join(stage, "api"))
	for _, file := range filenames {
		_ = parent.Remove(filepath.Join(stage, file))
	}
	_ = parent.Remove(stage)
}

// A producer receives only token/certificate, not the device enrollment or TLS key.
func prepareProducer(root *os.Root, stage string, c Config) error {
	directory := filepath.Join(stage, "api")
	if err := root.Mkdir(directory, 0o700); err != nil {
		return err
	}
	for _, name := range []string{"api-token", "tls.crt"} {
		if err := root.Link(filepath.Join(stage, name), filepath.Join(directory, name)); err != nil {
			return err
		}
	}
	if err := root.Chown(directory, c.UID, c.GID); err != nil {
		return err
	}
	return syncDirectory(root, directory)
}

func writePrivate(root *os.Root, name string, data []byte, c Config) error {
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	if _, err = file.Write(data); err != nil {
		return err
	}
	if err = file.Chown(c.UID, c.GID); err != nil {
		return err
	}
	return file.Sync()
}

func syncDirectory(root *os.Root, name string) error {
	file, err := root.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	return file.Sync()
}

func validateExisting(parent *os.Root, name string, c Config, enrollment []byte) error {
	info, err := parent.Lstat(name)
	if err != nil || !privateDirectory(info, c) {
		return errors.New("setup: existing destination must be a private owned directory")
	}
	root, err := parent.OpenRoot(name)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	files := make(map[string][]byte)
	for _, file := range filenames {
		files[file], err = readExisting(root, file, c)
		if err != nil {
			return err
		}
		defer clear(files[file])
	}
	if !bytes.Equal(files["device.json"], enrollment) || len(bytes.TrimSpace(files["api-token"])) < 32 {
		return errors.New("setup: enrollment differs or token is invalid; existing credentials preserved")
	}
	if err = validateProducer(root, c, files); err != nil {
		return err
	}
	return validateTLS(files)
}

func validateProducer(root *os.Root, c Config, files map[string][]byte) error {
	info, err := root.Lstat("api")
	if err != nil || !privateDirectory(info, c) {
		return errors.New("setup: unsafe producer credentials")
	}
	api, err := root.OpenRoot("api")
	if err != nil {
		return err
	}
	defer func() { _ = api.Close() }()
	c.Directory = filepath.Join(c.Directory, "api")
	for _, name := range []string{"api-token", "tls.crt"} {
		data, err := readExisting(api, name, c)
		if err != nil || !bytes.Equal(data, files[name]) {
			return errors.New("setup: producer credentials differ or are incomplete")
		}
	}
	return nil
}

func privateDirectory(info os.FileInfo, c Config) bool {
	return info.IsDir() && info.Mode().Perm() == 0o700 && owned(info, c)
}

func readExisting(root *os.Root, name string, c Config) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !owned(info, c) {
		return nil, errors.New("setup: incomplete or unsafe credentials; restore backup, do not rotate silently")
	}
	return hostprovision.ReadPrivateFile(filepath.Join(c.Directory, name), 4096)
}

func owned(info os.FileInfo, c Config) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && (c.UID == -1 || int(stat.Uid) == c.UID) && (c.GID == -1 || int(stat.Gid) == c.GID)
}
