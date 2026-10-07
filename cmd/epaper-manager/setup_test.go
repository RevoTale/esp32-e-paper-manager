package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
)

func TestSetupCLI(t *testing.T) {
	parent := t.TempDir()
	source := filepath.Join(parent, "device.json")
	if err := hostprovision.SaveEnrollment(source, hostprovision.Enrollment{
		DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Timezone: "Europe/Kyiv",
	}); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(parent, "credentials")
	if err := run([]string{"setup", "-enrollment", source, "-output", output}); err != nil {
		t.Fatal(err)
	}
	if _, err := readSecret(filepath.Join(output, "api-token")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(output, "tls.crt")); err != nil {
		t.Fatal(err)
	}
	for _, arguments := range [][]string{
		{"-unknown"}, {}, {"-enrollment", source},
		{"-enrollment", source, "-output", output, "extra"},
		{"-enrollment", source + ".missing", "-output", output},
	} {
		if err := runSetup(arguments); err == nil {
			t.Fatal("invalid setup accepted")
		}
	}
}
