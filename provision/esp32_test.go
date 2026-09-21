package provision

import "testing"

func TestESP32CodecDoesNotRelaxPicoPolicy(t *testing.T) {
	config := validConfig()
	config.Auth = AuthWPA2PSK
	codec := ESP32Codec()
	if config.Validate() == nil || config.ValidateFor(codec) != nil {
		t.Fatal("WPA2 must require explicit ESP32 policy")
	}
	request := Request{Operation: OperationProvision, Config: config}
	wire := make([]byte, RequestSize)
	if err := codec.EncodeRequest(wire, request); err != nil {
		t.Fatal(err)
	}
	if wire[16] != 2 {
		t.Fatal("wrong auth encoding")
	}
	if _, err := DecodeRequest(wire); err == nil {
		t.Fatal("Pico accepted WPA2")
	}
	config.Auth = AuthWPA3SAE
	if config.ValidateFor(codec) == nil {
		t.Fatal("ESP32 must not silently reinterpret WPA3")
	}
}

func TestESP32ResponseDoesNotRelaxPicoPolicy(t *testing.T) {
	config := validConfig()
	config.Auth = AuthWPA2PSK
	codec := ESP32Codec()
	wire := make([]byte, ResponseSize)
	response := Response{Operation: OperationInspect, State: StateProvisioned, Generation: 1,
		Auth: config.Auth, DeviceID: config.DeviceID, SSID: config.SSID, Manager: config.Manager, Timezone: config.Timezone}
	if err := codec.EncodeResponse(wire, response); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeResponse(wire); err == nil {
		t.Fatal("default response accepted WPA2")
	}
	if got, err := codec.DecodeResponse(wire); err != nil || got != response {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestESP32CodecRejectsUnsupportedIPv6AndUnknownAuth(t *testing.T) {
	config := validConfig()
	config.Auth = AuthWPA2PSK
	config.Manager = "tcp://[2001:db8::1]:443"
	if config.ValidateFor(ESP32Codec()) == nil {
		t.Fatal("native IPv6 not implemented")
	}
	config.Manager = "tcp://manager:000000000000000000001"
	if config.ValidateFor(ESP32Codec()) != nil {
		t.Fatal("valid numeric port")
	}
	config.Auth = 99
	if config.ValidateFor(ESP32Codec()) == nil {
		t.Fatal("unknown auth")
	}
}
