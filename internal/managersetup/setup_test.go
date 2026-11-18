package managersetup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
)

func TestSetupCreatesAndPreservesCredentials(t *testing.T) {
	configuration := fixture(t)
	if err := Run(configuration); err != nil {
		t.Fatal(err)
	}
	before := map[string]string{}
	for _, name := range []string{"device.json", "api-token", "tls.crt", "tls.key"} {
		path := filepath.Join(configuration.Directory, name)
		data, err := hostprovision.ReadPrivateFile(path, 4096)
		if err != nil {
			t.Fatal(err)
		}
		before[name] = string(data)
	}
	if err := Run(configuration); err != nil {
		t.Fatal(err)
	}
	for name, original := range before {
		data, err := os.ReadFile(filepath.Join(configuration.Directory, name))
		if err != nil || string(data) != original {
			t.Fatalf("setup changed %s: %v", name, err)
		}
	}
}
