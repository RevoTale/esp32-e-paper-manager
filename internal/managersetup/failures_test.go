package managersetup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
)

func fixture(t *testing.T) Config {
	t.Helper()
	parent := t.TempDir()
	c := Config{Enrollment: filepath.Join(parent, "source.json"), Directory: filepath.Join(parent, "credentials"), UID: -1, GID: -1}
	if err := hostprovision.SaveEnrollment(c.Enrollment, hostprovision.Enrollment{
		DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Timezone: "Europe/Kyiv",
	}); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestInvalidInputDoesNotCreateDestination(t *testing.T) {
	for _, scenario := range []string{"empty", "owner", "missing-source", "invalid-source", "missing-parent", "public-parent", "root"} {
		t.Run(scenario, func(t *testing.T) {
			c := fixture(t)
			switch scenario {
			case "empty":
				c.Directory = ""
			case "owner":
				c.UID = -2
			case "missing-source":
				c.Enrollment += ".missing"
			case "invalid-source":
				mustWrite(t, c.Enrollment, []byte("{}"))
			case "missing-parent":
				c.Directory = filepath.Join(c.Directory, "missing", "child")
			case "public-parent":
				mustChmod(t, filepath.Dir(c.Directory), 0o777)
			case "root":
				c.Directory = "/"
			}
			if err := Run(c); err == nil {
				t.Fatal("unsafe input accepted")
			}
		})
	}
}

func TestLongNameAndEntropyFailureLeaveNoDestination(t *testing.T) {
	c := fixture(t)
	c.Directory = filepath.Join(filepath.Dir(c.Directory), strings.Repeat("x", 300))
	if err := Run(c); err == nil {
		t.Fatal("long filename accepted")
	}
	c = fixture(t)
	c.random = bytes.NewReader(nil)
	if err := Run(c); err == nil {
		t.Fatal("entropy failure accepted")
	}
	if _, err := os.Lstat(c.Directory); !os.IsNotExist(err) {
		t.Fatal("entropy failure created credentials", err)
	}
}

func TestPublicationDoesNotReplaceExistingDirectory(t *testing.T) {
	c := fixture(t)
	if err := Run(c); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(filepath.Dir(c.Directory))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err = create(root, "credentials", c, []byte("different")); err == nil {
		t.Fatal("publication replaced an existing directory")
	}
	if err = Run(c); err != nil {
		t.Fatal("failed publication corrupted original credentials", err)
	}
	entries, err := root.ReadFile("credentials/device.json")
	if err != nil || bytes.Equal(entries, []byte("different")) {
		t.Fatal("enrollment changed", err)
	}
}

func TestExistingCredentialsAreNeverRepairedOrRotated(t *testing.T) {
	for _, scenario := range []string{"missing", "public", "directory-mode", "symlink", "enrollment", "token", "tls", "owner", "oversized", "api-directory", "api-token"} {
		t.Run(scenario, func(t *testing.T) {
			c := fixture(t)
			if err := Run(c); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(c.Directory, "api-token")
			switch scenario {
			case "api-directory":
				mustChmod(t, filepath.Join(c.Directory, "api"), 0o755)
			case "api-token":
				removeOrLink(t, filepath.Join(c.Directory, "api", "api-token"), "", false)
			default:
				mutate(t, scenario, path, &c)
			}
			if err := Run(c); err == nil {
				t.Fatal("corrupted destination accepted")
			}
		})
	}
}

func mutate(t *testing.T, scenario, path string, c *Config) {
	t.Helper()
	switch scenario {
	case "missing", "symlink":
		removeOrLink(t, path, c.Enrollment, scenario == "symlink")
	case "public":
		mustChmod(t, path, 0o644)
	case "directory-mode":
		mustChmod(t, c.Directory, 0o755)
	case "enrollment":
		data, err := os.ReadFile(c.Enrollment)
		if err != nil {
			t.Fatal(err)
		}
		mustWrite(t, c.Enrollment, append(data, '\n'))
	case "token":
		mustWrite(t, path, []byte("short"))
	case "tls":
		mustWrite(t, filepath.Join(c.Directory, "tls.key"), []byte("broken"))
	case "owner":
		c.UID = os.Getuid() + 1
	case "oversized":
		mustWrite(t, path, make([]byte, 4097))
	}
}

func removeOrLink(t *testing.T, path, target string, link bool) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if link {
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDestinationSymlinkAndIncompleteDirectory(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		c := fixture(t)
		var err error
		if symlink {
			err = os.Symlink(filepath.Dir(c.Enrollment), c.Directory)
		} else {
			err = os.Mkdir(c.Directory, 0o700)
		}
		if err != nil {
			t.Fatal(err)
		}
		if err = Run(c); err == nil {
			t.Fatal("unsafe destination accepted")
		}
	}
}

func TestClosedRootOperationsFail(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = root.Close(); err != nil {
		t.Fatal(err)
	}
	if err = writePrivate(root, "token", nil, Config{UID: -1, GID: -1}); err == nil {
		t.Fatal("write accepted closed root")
	}
	if err = syncDirectory(root, "."); err == nil {
		t.Fatal("sync accepted closed root")
	}
	if err = create(root, "credentials", Config{UID: -1, GID: -1}, nil); err == nil {
		t.Fatal("create accepted closed root")
	}
	if err = prepareProducer(root, "stage", Config{UID: -1, GID: -1}); err == nil {
		t.Fatal("producer creation accepted closed root")
	}
}

func TestProducerLinksRequireSourceFiles(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err = root.Mkdir("stage", 0o700); err != nil {
		t.Fatal(err)
	}
	if err = prepareProducer(root, "stage", Config{UID: -1, GID: -1}); err == nil {
		t.Fatal("producer prepared without source files")
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustChmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}
