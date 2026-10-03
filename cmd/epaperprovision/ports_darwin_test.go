//go:build darwin

package main

import (
	"strings"
	"testing"

	"go.bug.st/serial"
)

func TestDarwinAutoRejectsBeforeOpeningPort(t *testing.T) {
	original := openSerial
	t.Cleanup(func() { openSerial = original })
	openSerial = func(string, *serial.Mode) (serialDevice, error) {
		t.Fatal("automatic discovery must not open a guessed port")
		return nil, nil
	}
	port, closePort, err := connect("auto")
	if err == nil || port != nil || !strings.Contains(err.Error(), "-port") || !strings.Contains(err.Error(), "epaperctl -list") {
		t.Fatalf("missing actionable explicit-port policy: %v", err)
	}
	closePort()
}
