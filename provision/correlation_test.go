package provision

import (
	"encoding/binary"
	"hash/crc32"
	"testing"
)

func TestCorrelatedESP32ReplyRejectsStaleIDs(t *testing.T) {
	codec := ESP32Codec()
	id := RequestID{1}
	var request, reply [512]byte
	if err := codec.EncodeCorrelatedRequest(request[:], Request{Operation: OperationInspect}, id); err != nil {
		t.Fatal(err)
	}
	if request[4] != 3 || request[486] != 1 {
		t.Fatal("missing request correlation")
	}
	if err := codec.EncodeResponse(reply[:], Response{Operation: OperationInspect, State: StateBlank}); err != nil {
		t.Fatal(err)
	}
	reply[4], reply[392] = 3, 1
	binary.BigEndian.PutUint32(reply[508:], crc32.ChecksumIEEE(reply[:508]))
	if _, err := codec.DecodeCorrelatedResponse(reply[:], id); err != nil {
		t.Fatal(err)
	}
	if _, err := codec.DecodeCorrelatedResponse(reply[:], RequestID{2}); err == nil {
		t.Fatal("stale reply accepted")
	}
	for _, index := range []int{4, 391, 408, 507} {
		bad := reply
		bad[index] ^= 1
		binary.BigEndian.PutUint32(bad[508:], crc32.ChecksumIEEE(bad[:508]))
		if _, err := codec.DecodeCorrelatedResponse(bad[:], id); err == nil {
			t.Fatalf("invalid byte %d accepted", index)
		}
	}
	reply[508] ^= 1
	if _, err := codec.DecodeCorrelatedResponse(reply[:], id); err == nil {
		t.Fatal("invalid CRC accepted")
	}
}

func TestCorrelationRequiresESP32AndNonzeroID(t *testing.T) {
	var data [512]byte
	request := Request{Operation: OperationInspect}
	for _, tc := range []struct {
		codec Codec
		id    RequestID
	}{{Codec{}, RequestID{1}}, {ESP32Codec(), RequestID{}}} {
		if err := tc.codec.EncodeCorrelatedRequest(data[:], request, tc.id); err == nil {
			t.Fatal("invalid policy or ID accepted")
		}
	}
	if err := ESP32Codec().EncodeCorrelatedRequest(data[:10], request, RequestID{1}); err == nil {
		t.Fatal("short buffer accepted")
	}
	if _, err := ESP32Codec().DecodeCorrelatedResponse(data[:10], RequestID{1}); err == nil {
		t.Fatal("short response accepted")
	}
}
