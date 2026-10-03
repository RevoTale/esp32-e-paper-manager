// Package screenwire defines EPS2 bounded screen records, not authentication.
// Network records must pass the authenticated encryption layer before decoding.
// EPS1 remains a separate historical package and is never auto-detected here.
package screenwire

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

const (
	HeaderSize       = 32
	MaxPayload       = 1024
	MaxRecord        = HeaderSize + MaxPayload
	StatusSize       = 48
	CapabilitiesSize = 40
)

type Kind uint8

const (
	Hello Kind = iota + 1
	Acquire
	Bind
	Begin
	Data
	Commit
	Query
	Abort
	Reply
	Health       // Additive request; Reply's existing wire value remains 9.
	DataPacked   // Negotiated PackBits; coordinates and progress are decoded bytes.
	PanelTrace   // Cached physical-cycle observation; no ownership or pixel proof.
	BeginRefresh // Negotiated per-transaction operator cadence, never a BUSY override.
	BeginRegion  // Old/new region contract; dispatch requires a capable physical adapter.
)

var ErrRecord = errors.New("screen wire: invalid EPS2 record")

type Record struct {
	Kind      Kind
	Pass      uint8
	Epoch, ID uint64
	Offset    uint32
	Payload   []byte
}

func (r Record) validShape(n int) bool {
	if (!r.Kind.request() && r.Kind != Reply) || n < 0 || n > MaxPayload {
		return false
	}
	rule := shapes[r.Kind]
	return rule.epoch.accepts(r.Epoch) && rule.id.accepts(r.ID) && r.validCoordinates() && r.validLength(rule.length, n)
}

// Encode borrows Payload only during this call; it must not overlap dst.
func Encode(dst []byte, r Record) (int, error) {
	n := HeaderSize + len(r.Payload)
	if len(dst) < n || !r.validShape(len(r.Payload)) {
		return 0, ErrRecord
	}
	clear(dst[:HeaderSize])
	copy(dst, "EPS2")
	dst[4], dst[5] = byte(r.Kind), r.Pass
	binary.LittleEndian.PutUint16(dst[6:8], uint16(len(r.Payload)))
	binary.LittleEndian.PutUint64(dst[8:16], r.Epoch)
	binary.LittleEndian.PutUint64(dst[16:24], r.ID)
	binary.LittleEndian.PutUint32(dst[24:28], r.Offset)
	copy(dst[HeaderSize:n], r.Payload)
	binary.LittleEndian.PutUint32(dst[28:32], checksum(dst[:n]))
	return n, nil
}

// Size rejects malformed header shapes before allocating/reading payload bytes.
func Size(header []byte) (int, error) {
	if len(header) < HeaderSize || string(header[:4]) != "EPS2" {
		return 0, ErrRecord
	}
	n := int(binary.LittleEndian.Uint16(header[6:8]))
	if !fields(header).validShape(n) {
		return 0, ErrRecord
	}
	return HeaderSize + n, nil
}

// Decode borrows src. Consume all fields before reusing the read buffer.
func Decode(src []byte) (Record, error) {
	n, err := Size(src)
	if err != nil || len(src) != n {
		return Record{}, ErrRecord
	}
	if binary.LittleEndian.Uint32(src[28:32]) != checksum(src) {
		return Record{}, ErrRecord
	}
	r := fields(src)
	r.Payload = src[HeaderSize:]
	return r, nil
}

func fields(src []byte) Record {
	return Record{Kind: Kind(src[4]), Pass: src[5], Epoch: binary.LittleEndian.Uint64(src[8:16]),
		ID: binary.LittleEndian.Uint64(src[16:24]), Offset: binary.LittleEndian.Uint32(src[24:28])}
}

func checksum(src []byte) uint32 {
	crc := crc32.Update(0, crc32.IEEETable, src[:28])
	return crc32.Update(crc, crc32.IEEETable, src[HeaderSize:])
}
