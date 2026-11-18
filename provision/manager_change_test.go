package provision

import (
	"encoding/binary"
	"hash/crc32"
	"strings"
	"testing"
)

func TestManagerChangeWire(t *testing.T) {
	change := ManagerChange{DeviceID: [16]byte{8}, Generation: 14, Endpoint: "tcp://paper.home.arpa:9757"}
	request := Request{Operation: OperationSetManager, Change: change}
	id := RequestID{9}
	codec := ESP32Codec()
	var record, expected [512]byte
	copy(expected[:], "EPCQ")
	expected[4], expected[5], expected[15], expected[21], expected[24], expected[486] = 3, 6, 14, byte(len(change.Endpoint)), 8, 9
	copy(expected[72:], change.Endpoint)
	binary.BigEndian.PutUint32(expected[508:], crc32.ChecksumIEEE(expected[:508]))
	if err := codec.EncodeCorrelatedRequest(record[:], request, id); err != nil || record != expected {
		t.Fatal("unexpected USB record", err)
	}
	if codec.EncodeRequest(record[:], request) == nil || (Codec{}).EncodeCorrelatedRequest(record[:], request, id) == nil {
		t.Fatal("endpoint mutation escaped ESP32 v3")
	}
	for _, bad := range []ManagerChange{{}, {DeviceID: change.DeviceID, Endpoint: change.Endpoint},
		{Generation: 1, Endpoint: change.Endpoint}, {DeviceID: change.DeviceID, Generation: 1, Endpoint: "https://example.org"}} {
		if err := bad.encode(record[:]); err == nil {
			t.Fatal("invalid change accepted")
		}
	}
	if change.encode(record[:511]) == nil {
		t.Fatal("accepted short buffer")
	}
}

func TestESP32EndpointForms(t *testing.T) {
	for _, endpoint := range []string{"tcp://paper.home.arpa:9757", "tcp://192.0.2.20:9757"} {
		if !ValidESP32Endpoint(endpoint) {
			t.Fatal(endpoint)
		}
	}
	for _, endpoint := range []string{"tcp://[::1]:9757", "tcp://paper:9757?", "tcp://paper:9757#", "TCP://paper:1",
		"tcp://paper:0", "tcp://paper:9757/", "tcp://192.168.1.999:9757", "tcp://192.168.01.2:9757", "tcp://127.1:9757"} {
		if ValidESP32Endpoint(endpoint) {
			t.Fatal(endpoint)
		}
	}
}

func TestPinnedEndpointRoundTrip(t *testing.T) {
	pin := strings.Repeat("ab", 32)
	for _, address := range []string{"paper.example:443", "192.0.2.18:9443"} {
		value := "tls://" + address + "#" + pin
		if !ESP32Codec().validManager(value) || (Codec{}).validManager(value) {
			t.Fatal(value)
		}
		change := ManagerChange{DeviceID: [16]byte{1}, Generation: 7, Endpoint: value}
		var page [RequestSize]byte
		if err := change.encode(page[:]); err != nil || string(page[72:72+len(value)]) != value {
			t.Fatal(err)
		}
	}
	for _, bad := range []string{"tls://paper:443", "tls://paper:443#" + strings.Repeat("00", 32), "tls://bad:443#abc", strings.Repeat("x", 256)} {
		if ValidESP32Endpoint(bad) {
			t.Fatal(bad)
		}
	}
}
