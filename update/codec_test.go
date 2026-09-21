package update

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"os"
	"strings"
	"testing"
)

func TestCodecMatchesIndependentV1Golden(t *testing.T) {
	hexWire, err := os.ReadFile("../testdata/update-v1-request.hex")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	wire, err := hex.DecodeString(strings.TrimSpace(string(hexWire)))
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}
	decoded, err := Decode(wire)
	if err != nil {
		t.Fatalf("Decode(golden) error = %v", err)
	}
	request := testRequest(t)
	if !request.SameIntent(decoded) {
		t.Fatal("golden request differs from documented v1 intent")
	}
	encoded := make([]byte, EncodedSize(request))
	n, err := Encode(encoded, request)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if !bytes.Equal(encoded[:n], wire) {
		t.Fatal("Encode() differs from independent v1 golden")
	}
}

func TestCodecRoundTripBorrowsWirePayload(t *testing.T) {
	request := testRequest(t)
	wire := make([]byte, EncodedSize(request))
	n, err := Encode(wire, request)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	decoded, err := Decode(wire[:n])
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !request.SameIntent(decoded) {
		t.Fatal("decoded request differs from encoded intent")
	}
	wire[HeaderSize] = 'X'
	if decoded.Document().HTML()[0] != 'X' {
		t.Fatal("Decode() copied payload")
	}
}

func TestCodecAcceptsMaximumTimezoneLength(t *testing.T) {
	base := testRequest(t)
	timezone := strings.Repeat("A", MaxTimezoneLength)
	request, err := NewRequest(base.ID(), base.UTC(), base.DisplayTime(), timezone, base.Document())
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	wire := make([]byte, EncodedSize(request))
	n, err := Encode(wire, request)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	decoded, err := Decode(wire[:n])
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if decoded.Timezone() != timezone {
		t.Fatalf("Timezone() length = %d, want %d", len(decoded.Timezone()), len(timezone))
	}
}

func TestCodecRejectsCorruptionWithoutReturningDocument(t *testing.T) {
	request := testRequest(t)
	wire := make([]byte, EncodedSize(request))
	n, err := Encode(wire, request)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	tests := []struct {
		name   string
		offset int
	}{
		{name: "header", offset: 20},
		{name: "payload", offset: HeaderSize},
		{name: "trailer", offset: n - 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			corrupt := bytes.Clone(wire[:n])
			corrupt[test.offset] ^= 0x01
			decoded, err := Decode(corrupt)
			if !errors.Is(err, ErrCodec) {
				t.Fatalf("Decode() error = %v, want ErrCodec", err)
			}
			if len(decoded.Document().HTML()) != 0 {
				t.Fatal("Decode() returned document on corruption")
			}
		})
	}
}

func TestCodecRejectsBoundsAndTrailingData(t *testing.T) {
	request := testRequest(t)
	wire := make([]byte, EncodedSize(request))
	if _, err := Encode(wire, Request{}); err == nil {
		t.Fatal("Encode(zero request) error = nil")
	}
	if _, err := Encode(wire[:len(wire)-1], request); !errors.Is(err, ErrBuffer) {
		t.Fatalf("Encode(short) error = %v, want ErrBuffer", err)
	}
	n, err := Encode(wire, request)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if _, err := Decode(wire[:HeaderSize-1]); !errors.Is(err, ErrCodec) {
		t.Fatalf("Decode(short) error = %v, want ErrCodec", err)
	}
	withTrailing := append(bytes.Clone(wire[:n]), 0)
	if _, err := Decode(withTrailing); !errors.Is(err, ErrCodec) {
		t.Fatalf("Decode(trailing) error = %v, want ErrCodec", err)
	}
}

func TestCodecRejectsUnsupportedOrMalformedMetadata(t *testing.T) {
	request := testRequest(t)
	wire := make([]byte, EncodedSize(request))
	n, err := Encode(wire, request)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	tests := []struct {
		name   string
		offset int
		value  byte
	}{
		{name: "envelope version", offset: 4, value: Version + 1},
		{name: "document profile", offset: 5, value: 0xff},
		{name: "document version", offset: 6, value: 0xff},
		{name: "reserved byte", offset: 7, value: 1},
		{name: "display time", offset: 72, value: 'X'},
		{name: "timezone", offset: 92, value: '/'},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := bytes.Clone(wire[:n])
			candidate[test.offset] = test.value
			rechecksum(candidate)
			if _, err := Decode(candidate); !errors.Is(err, ErrCodec) {
				t.Fatalf("Decode() error = %v, want ErrCodec", err)
			}
		})
	}
}

func rechecksum(wire []byte) {
	binary.BigEndian.PutUint32(wire[156:160], crc32.ChecksumIEEE(wire[:156]))
	payloadEnd := len(wire) - TrailerSize
	binary.BigEndian.PutUint32(wire[payloadEnd:], crc32.ChecksumIEEE(wire[:payloadEnd]))
}
