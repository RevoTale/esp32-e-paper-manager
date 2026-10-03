// Package provision defines USB-only network configuration and its flash journal.
package provision

import (
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	MaxSSID       = 32
	MaxPassphrase = 63
	MaxManager    = 255
	MaxTimezone   = 64
)

var ErrInvalidConfig = errors.New("provision: invalid configuration")

type AuthMode uint8

const AuthWPA3SAE AuthMode = 1
const AuthWPA2PSK AuthMode = 2

type Config struct {
	SSID       string
	Passphrase string
	Manager    string
	Timezone   string
	DeviceID   [16]byte
	DeviceKey  [32]byte
	Auth       AuthMode
}

func (c Config) Validate() error {
	return c.ValidateFor(Codec{})
}

func (c Config) ValidateFor(codec Codec) error {
	if c.Auth != codec.AuthMode() || !boundedText(c.SSID, 1, MaxSSID) ||
		!boundedText(c.Passphrase, 8, MaxPassphrase) || !codec.validManager(c.Manager) ||
		!validTimezone(c.Timezone) || allZero(c.DeviceID[:]) || allZero(c.DeviceKey[:]) {
		return ErrInvalidConfig
	}
	return nil
}

func (codec Codec) validManager(value string) bool {
	// Native ESP32 IPv6 is deferred; apply the same policy to public metadata.
	return validManager(value) && (!codec.esp32 || !strings.Contains(value, "["))
}

func boundedText(value string, minimum, maximum int) bool {
	if len(value) < minimum || len(value) > maximum || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

func validManager(value string) bool {
	if len(value) == 0 || len(value) > MaxManager {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || !managerURLShape(parsed) {
		return false
	}
	host, port, err := net.SplitHostPort(parsed.Host)
	if err != nil || !validHost(host) {
		return false
	}
	number, err := strconv.Atoi(port)
	return err == nil && number > 0 && number <= 65535
}

func managerURLShape(parsed *url.URL) bool {
	return parsed.Scheme == "tcp" && parsed.User == nil && parsed.Path == "" &&
		parsed.RawQuery == "" && parsed.Fragment == ""
}

func validHost(host string) bool {
	if _, err := netip.ParseAddr(host); err == nil {
		return true
	}
	if len(host) == 0 || len(host) > 253 || strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if !validDNSLabel(label) {
			return false
		}
	}
	return true
}

func validDNSLabel(label string) bool {
	if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for _, character := range label {
		if !dnsCharacter(character) {
			return false
		}
	}
	return true
}

func dnsCharacter(character rune) bool {
	return character == '-' || character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' || character >= '0' && character <= '9'
}

func validTimezone(value string) bool {
	if !boundedText(value, 1, MaxTimezone) || value[0] == '/' || value[len(value)-1] == '/' ||
		strings.Contains(value, "..") || strings.Contains(value, "//") {
		return false
	}
	for _, character := range value {
		if !timezoneCharacter(character) {
			return false
		}
	}
	return true
}

func timezoneCharacter(character rune) bool {
	return strings.ContainsRune("_+-./", character) || character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' || character >= '0' && character <= '9'
}

func allZero(value []byte) bool {
	var combined byte
	for _, item := range value {
		combined |= item
	}
	return combined == 0
}
