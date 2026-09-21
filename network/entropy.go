package network

import (
	"encoding/binary"
	"errors"
)

var ErrEntropy = errors.New("network: entropy unavailable")

type WordSource func() (uint32, error)

// CounterReader mixes a monotonic counter into platform-provided random words.
// The manager-provided cryptographic challenge remains the security anchor;
// this reader guarantees per-boot nonce variation on the Pico side.
type CounterReader struct {
	source  WordSource
	counter uint64
}

func NewCounterReader(source WordSource, counter uint64) (*CounterReader, error) {
	if source == nil {
		return nil, ErrEntropy
	}
	return &CounterReader{source: source, counter: counter}, nil
}

func (reader *CounterReader) Read(destination []byte) (int, error) {
	if reader == nil || reader.source == nil {
		return 0, ErrEntropy
	}
	for offset := 0; offset < len(destination); offset += 4 {
		value, err := reader.source()
		if err != nil {
			return offset, err
		}
		reader.counter++
		value ^= uint32(reader.counter) ^ uint32(reader.counter>>32)
		limit := min(4, len(destination)-offset)
		var wire [4]byte
		binary.BigEndian.PutUint32(wire[:], value)
		copy(destination[offset:offset+limit], wire[:limit])
	}
	return len(destination), nil
}
