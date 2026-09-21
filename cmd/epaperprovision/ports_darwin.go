//go:build darwin

package main

import "errors"

// Detailed macOS USB metadata in serial/enumerator requires Cgo. Port names
// alone cannot establish USB identity; follow epaperctl's explicit-port policy.
func findPort() (string, error) {
	return "", errors.New("automatic USB identity discovery is unavailable in this pure-Go macOS build; pass -port with the exact /dev/cu.usbmodem path (list names with epaperctl -list)")
}
