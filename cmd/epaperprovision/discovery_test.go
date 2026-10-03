//go:build !darwin

package main

import (
	"strings"
	"testing"

	"go.bug.st/serial/enumerator"
)

func TestDiscoveryRejectsMultipleMatchingPicos(t *testing.T) {
	original := enumeratePorts
	t.Cleanup(func() { enumeratePorts = original })
	enumeratePorts = func(...func(string, string) bool) ([]*enumerator.PortDetails, error) {
		return []*enumerator.PortDetails{
			{Name: "first", IsUSB: true, VID: tinyGoVID, PID: tinyGoPID},
			{Name: "second", IsUSB: true, VID: tinyGoVID, PID: tinyGoPID},
		}, nil
	}
	if name, err := findPort(); err == nil || name != "" || !strings.Contains(err.Error(), "-port") {
		t.Fatalf("ambiguous discovery selected %q: %v", name, err)
	}
}

func TestDiscoveryUsesUSBIdentityNotPathOrHexCase(t *testing.T) {
	original := enumeratePorts
	t.Cleanup(func() { enumeratePorts = original })
	enumeratePorts = func(...func(string, string) bool) ([]*enumerator.PortDetails, error) {
		return []*enumerator.PortDetails{
			{Name: "not-usb", VID: tinyGoVID, PID: tinyGoPID},
			{Name: "other-vendor", IsUSB: true, VID: "FFFF", PID: tinyGoPID},
			{Name: "other-product", IsUSB: true, VID: tinyGoVID, PID: "FFFF"},
			{Name: "identified", IsUSB: true, VID: "2e8a", PID: "000a"},
		}, nil
	}
	if name, err := findPort(); err != nil || name != "identified" {
		t.Fatalf("USB identity selection %q: %v", name, err)
	}
}
