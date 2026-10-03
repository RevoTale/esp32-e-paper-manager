package provision

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"testing"
)

func TestEveryAuthByteRequiresItsExplicitCodec(t *testing.T) {
	for _, codec := range []Codec{{}, ESP32Codec()} {
		config := validConfig()
		config.Auth = codec.AuthMode()
		request := make([]byte, RequestSize)
		if err := codec.EncodeRequest(request, Request{Operation: OperationProvision, Config: config}); err != nil {
			t.Fatal(err)
		}
		response := make([]byte, ResponseSize)
		if err := codec.EncodeResponse(response, Response{Operation: OperationInspect, State: StateProvisioned,
			Generation: 1, Auth: config.Auth, DeviceID: config.DeviceID, SSID: config.SSID,
			Manager: config.Manager, Timezone: config.Timezone}); err != nil {
			t.Fatal(err)
		}
		for auth := 0; auth <= 255; auth++ {
			request[16], response[7] = byte(auth), byte(auth)
			binary.BigEndian.PutUint32(request[508:], crc32.ChecksumIEEE(request[:508]))
			binary.BigEndian.PutUint32(response[508:], crc32.ChecksumIEEE(response[:508]))
			_, requestErr := codec.DecodeRequest(request)
			_, responseErr := codec.DecodeResponse(response)
			want := AuthMode(auth) == codec.AuthMode()
			if (requestErr == nil) != want || (responseErr == nil) != want {
				t.Fatalf("codec auth=%d wire auth=%d request=%v response=%v", codec.AuthMode(), auth, requestErr, responseErr)
			}
		}
	}
}

func TestPicoFlashPolicyCannotBeRelaxedByESP32Codec(t *testing.T) {
	flash := newFakeFlash()
	store, err := NewStore(flash, 0)
	if err != nil {
		t.Fatal(err)
	}
	config := validConfig()
	if _, err = store.Save(config); err != nil {
		t.Fatal(err)
	}
	before := bytes.Clone(flash.data)
	config.Auth = AuthWPA2PSK
	if config.ValidateFor(ESP32Codec()) != nil {
		t.Fatal("ESP32 fixture invalid")
	}
	if _, err = store.Save(config); !errors.Is(err, ErrInvalidConfig) || !bytes.Equal(before, flash.data) {
		t.Fatal("Pico store mutated for WPA2", err)
	}
	flash = newFakeFlash()
	encodeRecord(flash.data[:RecordSize], config, 1)
	store, err = NewStore(flash, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.Load(); !errors.Is(err, ErrCorrupt) {
		t.Fatal("Pico flash decoding admitted WPA2", err)
	}
}
