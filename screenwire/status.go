package screenwire

import (
	"encoding/binary"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

type Code uint8

const (
	CodeOK Code = iota
	CodeRecord
	CodeLease
	CodeBusy
	CodeStale
	CodeConflict
	CodeCooldown
	CodeState
	CodeChunk
	CodeDigest
	CodeTimeout
	CodeHardware
	CodeConfig
)

type Domain uint8

const (
	DomainNone Domain = iota
	DomainPanel
)

// Diagnostic carries only bounded numeric hardware evidence, never err.Error().
// Panel-specific phase/step/code values belong to the negotiated profile.
type Diagnostic struct {
	Domain                     Domain
	Code, Phase, Step, Command uint8
	Offset                     int32
	BusyKnown, Busy            bool
}

type Status struct {
	Operation    Kind
	Code         Code
	State        streamrx.State
	Pass         uint8
	Offset       uint32
	CurrentImage bool
	Diagnostic   Diagnostic
	CooldownMS   uint32
	Generation   uint64
	Boot         [16]byte
}

func (s Status) valid() bool {
	return s.Operation.request() && s.Code <= CodeConfig && s.State <= streamrx.Closed && s.Pass <= 2 && s.validEvidence()
}

func (s Status) validEvidence() bool {
	if s.Boot == [16]byte{} {
		return false
	}
	if s.CurrentImage && s.State != streamrx.Complete {
		return false
	}
	return s.Diagnostic.valid() && s.validProgress()
}

func (s Status) validProgress() bool {
	if s.Pass == 2 && s.Offset != 0 {
		return false
	}
	switch s.State {
	case streamrx.Idle:
		return s.Pass == 0 && s.Offset == 0
	case streamrx.Receiving:
		return s.Pass < 2
	case streamrx.Ready, streamrx.Complete:
		return s.Pass >= 1 && s.Offset == 0
	default:
		return true
	}
}

func (d Diagnostic) valid() bool {
	if d.Domain == DomainNone {
		return d == Diagnostic{}
	}
	return d.Domain == DomainPanel && d.Code != 0 && d.Offset >= -1 && (!d.Busy || d.BusyKnown)
}

func EncodeStatus(dst []byte, s Status) error {
	if len(dst) != StatusSize || !s.valid() {
		return ErrRecord
	}
	clear(dst)
	dst[0], dst[1], dst[2], dst[3] = byte(s.Operation), byte(s.Code), byte(s.State), s.Pass
	binary.LittleEndian.PutUint32(dst[4:8], s.Offset)
	if s.CurrentImage {
		dst[8] = 1
	}
	d := s.Diagnostic
	dst[9], dst[10], dst[11], dst[12], dst[13] = byte(d.Domain), d.Code, d.Phase, d.Step, d.Command
	if d.BusyKnown {
		dst[14] |= 1
	}
	if d.Busy {
		dst[14] |= 2
	}
	binary.LittleEndian.PutUint32(dst[16:20], uint32(d.Offset))
	binary.LittleEndian.PutUint32(dst[20:24], s.CooldownMS)
	binary.LittleEndian.PutUint64(dst[24:32], s.Generation)
	copy(dst[32:48], s.Boot[:])
	return nil
}

func DecodeStatus(src []byte) (Status, error) {
	if len(src) != StatusSize {
		return Status{}, ErrRecord
	}
	if src[8] > 1 || src[14]&^byte(3) != 0 || src[15] != 0 {
		return Status{}, ErrRecord
	}
	s := Status{Operation: Kind(src[0]), Code: Code(src[1]), State: streamrx.State(src[2]), Pass: src[3],
		Offset: binary.LittleEndian.Uint32(src[4:8]), CurrentImage: src[8] != 0,
		CooldownMS: binary.LittleEndian.Uint32(src[20:24]), Generation: binary.LittleEndian.Uint64(src[24:32])}
	s.Diagnostic = Diagnostic{Domain: Domain(src[9]), Code: src[10], Phase: src[11], Step: src[12], Command: src[13],
		BusyKnown: src[14]&1 != 0, Busy: src[14]&2 != 0, Offset: int32(binary.LittleEndian.Uint32(src[16:20]))}
	copy(s.Boot[:], src[32:48])
	if !s.valid() {
		return Status{}, ErrRecord
	}
	return s, nil
}
