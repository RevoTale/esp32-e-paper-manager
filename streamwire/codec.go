// Package streamwire defines experimental USB-only EPS1 records. CRC detects
// corruption, not forgery. Never expose this framing as authenticated Wi-Fi.
package streamwire

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

const HeaderSize = 32
const MaxPayload = 100
const MaxRecord = HeaderSize + MaxPayload

type Kind uint8

const (
	Hello Kind = iota + 1
	Begin
	Data
	Commit
	Query
	Abort
	Reply
)

var ErrRecord = errors.New("stream wire: invalid record")

type Record struct {
	Kind      Kind
	Pass      uint8
	Epoch, ID uint64
	Offset    uint32
	Payload   []byte
}

// Encode borrows Payload. Decode aliases its input; consume before reusing it.
func Encode(dst []byte, r Record) (int, error) {
	n := HeaderSize + len(r.Payload)
	if len(dst) < n || len(r.Payload) > MaxPayload || r.Kind < Hello || r.Kind > Reply {
		return 0, ErrRecord
	}
	clear(dst[:n])
	copy(dst, "EPS1")
	dst[4], dst[5] = byte(r.Kind), r.Pass
	binary.LittleEndian.PutUint16(dst[6:8], uint16(len(r.Payload)))
	binary.LittleEndian.PutUint64(dst[8:16], r.Epoch)
	binary.LittleEndian.PutUint64(dst[16:24], r.ID)
	binary.LittleEndian.PutUint32(dst[24:28], r.Offset)
	copy(dst[HeaderSize:n], r.Payload)
	binary.LittleEndian.PutUint32(dst[28:32], checksum(dst[:n]))
	return n, nil
}

func Size(header []byte) (int, error) {
	if len(header) < HeaderSize || string(header[:4]) != "EPS1" {
		return 0, ErrRecord
	}
	n := int(binary.LittleEndian.Uint16(header[6:8]))
	if n > MaxPayload || Kind(header[4]) < Hello || Kind(header[4]) > Reply {
		return 0, ErrRecord
	}
	return HeaderSize + n, nil
}

func Decode(src []byte) (Record, error) {
	n, err := Size(src)
	if err != nil || len(src) != n {
		return Record{}, ErrRecord
	}
	if binary.LittleEndian.Uint32(src[28:32]) != checksum(src) {
		return Record{}, ErrRecord
	}
	return Record{Kind: Kind(src[4]), Pass: src[5], Epoch: binary.LittleEndian.Uint64(src[8:16]),
		ID: binary.LittleEndian.Uint64(src[16:24]), Offset: binary.LittleEndian.Uint32(src[24:28]), Payload: src[HeaderSize:]}, nil
}

func checksum(src []byte) uint32 {
	crc := crc32.Update(0, crc32.IEEETable, src[:28])
	return crc32.Update(crc, crc32.IEEETable, src[HeaderSize:])
}
