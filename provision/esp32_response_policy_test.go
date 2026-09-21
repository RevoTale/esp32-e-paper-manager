package provision

import (
	"encoding/binary"
	"hash/crc32"
	"testing"
)

func TestESP32ResponseUsesTheSameIPv4PolicyAsRequests(t *testing.T) {
	config := validConfig()
	response := Response{Operation: OperationInspect, State: StateProvisioned, Generation: 1,
		Auth: AuthWPA3SAE, DeviceID: config.DeviceID, SSID: config.SSID,
		Manager: "tcp://[2001:db8::1]:443", Timezone: config.Timezone}
	wire := make([]byte, ResponseSize)
	if err := EncodeResponse(wire, response); err != nil {
		t.Fatal("existing Pico IPv6 policy changed", err)
	}
	wire[7] = byte(AuthWPA2PSK)
	binary.BigEndian.PutUint32(wire[508:], crc32.ChecksumIEEE(wire[:508]))
	if _, err := ESP32Codec().DecodeResponse(wire); err == nil {
		t.Error("ESP32 response decoder accepted an unsupported IPv6 endpoint")
	}
	response.Auth = AuthWPA2PSK
	if err := ESP32Codec().EncodeResponse(wire, response); err == nil {
		t.Error("ESP32 response encoder accepted an unsupported IPv6 endpoint")
	}
}
