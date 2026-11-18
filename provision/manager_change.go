package provision

import (
	"encoding/binary"
	"encoding/hex"
	"net/netip"
	"strings"
)

// ManagerChange replaces one endpoint on the exact journal snapshot inspected over USB.
// It contains no Wi-Fi credentials or device key.
type ManagerChange struct {
	DeviceID   [16]byte
	Generation uint64
	Endpoint   string
}

// ValidESP32Endpoint admits the same literal TCP DNS/IPv4 endpoint as the receiver.
func ValidESP32Endpoint(endpoint string) bool {
	if len(endpoint) > MaxManager {
		return false
	}
	if address, secure := strings.CutPrefix(endpoint, "tls://"); secure {
		hostPort, fingerprint, found := strings.Cut(address, "#")
		pin, err := hex.DecodeString(fingerprint)
		if !found || err != nil || len(pin) != 32 || allZero(pin) {
			return false
		}
		endpoint = "tcp://" + hostPort
	}
	return validESP32TCP(endpoint)
}

func validESP32TCP(endpoint string) bool {
	address, ok := strings.CutPrefix(endpoint, "tcp://")
	if !ok || strings.ContainsAny(address, "?#%") || !validManager(endpoint) || strings.Contains(address, "[") {
		return false
	}
	host, _, _ := strings.Cut(address, ":")
	if strings.Trim(host, "0123456789.") != "" {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.Is4()
}

func (change ManagerChange) encode(dst []byte) error {
	if len(dst) < RequestSize || change.DeviceID == ([16]byte{}) || change.Generation == 0 ||
		!ValidESP32Endpoint(change.Endpoint) {
		return ErrInvalidConfig
	}
	clear(dst[:RequestSize])
	copy(dst, "EPCQ")
	dst[5] = byte(OperationSetManager)
	binary.BigEndian.PutUint64(dst[8:16], change.Generation)
	binary.BigEndian.PutUint16(dst[20:22], uint16(len(change.Endpoint)))
	copy(dst[24:40], change.DeviceID[:])
	copy(dst[72:327], change.Endpoint)
	return nil
}
