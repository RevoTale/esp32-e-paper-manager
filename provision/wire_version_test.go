package provision

import (
	"encoding/binary"
	"hash/crc32"
	"testing"
)

func TestUSBVersionRejectsLegacyBeforeMutation(t *testing.T) {
	var wire [RequestSize]byte
	if err := EncodeRequest(wire[:], Request{Operation: OperationProvision, Config: validConfig()}); err != nil {
		t.Fatal(err)
	}
	if wire[4] != 2 {
		t.Fatalf("USB version=%d; must reject old unfenced firmware before mutation", wire[4])
	}
	wire[4] = 1
	rechecksum(wire[:])
	if _, err := DecodeRequest(wire[:]); err == nil {
		t.Fatal("legacy request accepted")
	}
}

func TestRequestRejectsReservedAndHiddenBytes(t *testing.T) {
	for _, index := range []int{6, 8, 15, 22, 23, 486, 507} {
		var wire [RequestSize]byte
		if err := EncodeRequest(wire[:], Request{Operation: OperationProvision, Config: validConfig()}); err != nil {
			t.Fatal(err)
		}
		wire[index] = 1
		rechecksum(wire[:])
		if _, err := DecodeRequest(wire[:]); err == nil {
			t.Errorf("reserved byte %d accepted", index)
		}
	}
}

func TestResponseRejectsLegacyAndReservedBytes(t *testing.T) {
	for _, index := range []int{4, 13, 15, 40, 72, 327, 391, 507} {
		var wire [ResponseSize]byte
		if err := EncodeResponse(wire[:], Response{Operation: OperationInspect}); err != nil {
			t.Fatal(err)
		}
		wire[index] = 1
		rechecksum(wire[:])
		if _, err := DecodeResponse(wire[:]); err == nil {
			t.Errorf("legacy/reserved byte %d accepted", index)
		}
	}
}

func rechecksum(wire []byte) {
	binary.BigEndian.PutUint32(wire[508:512], crc32.ChecksumIEEE(wire[:508]))
}
