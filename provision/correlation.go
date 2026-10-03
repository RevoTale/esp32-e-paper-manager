package provision

import (
	"encoding/binary"
	"hash/crc32"
)

// RequestID binds a physical USB reply to one attempt, not a retryable intent.
// It is neither authentication nor a durable idempotency key.
type RequestID [16]byte

// EncodeCorrelatedRequest adds the ESP32 v3 envelope; stored EPC2 stays unchanged.
func (codec Codec) EncodeCorrelatedRequest(dst []byte, request Request, id RequestID) error {
	if !codec.esp32 || id == (RequestID{}) {
		return ErrInvalidConfig
	}
	if err := codec.EncodeRequest(dst, request); err != nil {
		return err
	}
	dst[4] = 3
	copy(dst[486:502], id[:])
	binary.BigEndian.PutUint32(dst[508:512], crc32.ChecksumIEEE(dst[:508]))
	return nil
}

// DecodeCorrelatedResponse validates the v3 envelope before applying all v2
// metadata and padding rules. A legacy/stale reply never acknowledges v3 work.
func (codec Codec) DecodeCorrelatedResponse(src []byte, id RequestID) (Response, error) {
	if !codec.esp32 || id == (RequestID{}) || len(src) != ResponseSize {
		return Response{}, ErrInvalidConfig
	}
	if string(src[:4]) != "EPCR" || src[4] != 3 || RequestID(src[392:408]) != id ||
		crc32.ChecksumIEEE(src[:508]) != binary.BigEndian.Uint32(src[508:]) {
		return Response{}, ErrInvalidConfig
	}
	var legacy [ResponseSize]byte
	copy(legacy[:], src)
	legacy[4] = 2
	clear(legacy[392:408])
	binary.BigEndian.PutUint32(legacy[508:], crc32.ChecksumIEEE(legacy[:508]))
	return codec.DecodeResponse(legacy[:])
}
