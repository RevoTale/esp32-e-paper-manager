package provision

// Codec selects one explicit board's admission policy. Its zero value retains
// Pico's WPA3-only contract; an ESP32 policy never affects device flash decoding.
type Codec struct{ esp32 bool }

func ESP32Codec() Codec { return Codec{esp32: true} }

func (c Codec) AuthMode() AuthMode {
	if c.esp32 {
		return AuthWPA2PSK
	}
	return AuthWPA3SAE
}

func EncodeRequest(dst []byte, request Request) error { return (Codec{}).EncodeRequest(dst, request) }
func EncodeResponse(dst []byte, response Response) error {
	return (Codec{}).EncodeResponse(dst, response)
}
func DecodeResponse(src []byte) (Response, error) { return (Codec{}).DecodeResponse(src) }
func DecodeRequest(src []byte) (Request, error)   { return (Codec{}).DecodeRequest(src) }
