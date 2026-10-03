package securetransport

import (
	"bytes"
	"testing"
)

// EPS2 carries a 32-byte header and at most 1024 authenticated pixel bytes.
// Use literal independent bounds: a test derived only from MaxPlaintext would
// also pass the old 284-byte configuration that cannot carry these records.
func TestScreenRecordEnvelopeCapacity(t *testing.T) {
	host, device := sessionPair(t)
	plain := bytes.Repeat([]byte{0x5a}, 1056)
	var envelope [1088]byte
	n, err := host.Seal(envelope[:], plain)
	if err != nil || n != 1088 {
		t.Fatal(n, err)
	}
	var decoded [1056]byte
	n, err = device.Open(decoded[:], envelope[:])
	if err != nil || n != 1056 || !bytes.Equal(decoded[:], plain) {
		t.Fatal(n, err)
	}
	if _, err = host.Seal(make([]byte, 1089), make([]byte, 1057)); err != ErrBounds {
		t.Fatal(err)
	}
}
