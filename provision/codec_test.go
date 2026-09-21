package provision

import (
	"bytes"
	"testing"
)

func TestRequestAndRedactedResponseRoundTrip(t *testing.T) {
	config := validConfig()
	requestWire := make([]byte, RequestSize)
	if err := EncodeRequest(requestWire, Request{Operation: OperationProvision, Config: config}); err != nil {
		t.Fatal(err)
	}
	request, err := DecodeRequest(requestWire)
	if err != nil || request.Config != config {
		t.Fatalf("request=%+v err=%v", request, err)
	}
	response := publicResponse(OperationInspect, config, 7)
	responseWire := make([]byte, ResponseSize)
	if err = EncodeResponse(responseWire, response); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(responseWire, []byte(config.Passphrase)) || bytes.Contains(responseWire, config.DeviceKey[:]) {
		t.Fatal("response exposed a secret")
	}
	decoded, err := DecodeResponse(responseWire)
	if err != nil || decoded != response {
		t.Fatalf("response=%+v err=%v", decoded, err)
	}
}

func TestCodecRejectsCorruptionAndInvalidOperations(t *testing.T) {
	wire := make([]byte, RequestSize)
	if err := EncodeRequest(wire, Request{Operation: OperationInspect}); err != nil {
		t.Fatal(err)
	}
	wire[10] ^= 1
	if _, err := DecodeRequest(wire); err == nil {
		t.Fatal("corrupt request accepted")
	}
	if err := EncodeRequest(wire, Request{Operation: 99}); err == nil {
		t.Fatal("invalid operation accepted")
	}
	response := make([]byte, ResponseSize)
	if err := EncodeResponse(response, Response{Operation: OperationInspect, State: 99}); err == nil {
		t.Fatal("invalid state accepted")
	}
	if _, err := DecodeResponse(response[:10]); err == nil {
		t.Fatal("short response accepted")
	}

}

func TestCodecRejectsShortDestinationsAndCorruptResponse(t *testing.T) {
	if err := EncodeRequest(make([]byte, 10), Request{Operation: OperationInspect}); err == nil {
		t.Fatal("short request destination accepted")
	}
	if _, err := DecodeRequest(make([]byte, 10)); err == nil {
		t.Fatal("short request accepted")
	}
	valid := make([]byte, ResponseSize)
	if err := EncodeResponse(valid, Response{Operation: OperationInspect, State: StateBlank}); err != nil {
		t.Fatal(err)
	}
	valid[20] ^= 1
	if _, err := DecodeResponse(valid); err == nil {
		t.Fatal("corrupt response accepted")
	}
	if err := EncodeResponse(make([]byte, 10), Response{Operation: OperationInspect, State: StateBlank}); err == nil {
		t.Fatal("short response destination accepted")
	}
}
