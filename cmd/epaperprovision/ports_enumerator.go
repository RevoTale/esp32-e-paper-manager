//go:build !darwin

package main

import (
	"errors"
	"strings"

	"go.bug.st/serial/enumerator"
)

var enumeratePorts = enumerator.GetDetailedPortsList

func findPort() (string, error) {
	ports, err := enumeratePorts()
	if err != nil {
		return "", err
	}
	match := ""
	for _, port := range ports {
		if port.IsUSB && strings.EqualFold(port.VID, tinyGoVID) && strings.EqualFold(port.PID, tinyGoPID) {
			if match != "" {
				return "", errors.New("multiple TinyGo Pico ports found; pass -port")
			}
			match = port.Name
		}
	}
	if match == "" {
		return "", errUSBNotFound
	}
	return match, nil
}
