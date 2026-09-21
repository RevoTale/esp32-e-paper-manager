package securetransport

import (
	"encoding/binary"
	"io"
	"time"
)

// ReadLimits separates idle waiting from one ciphertext record's total budget.
// SetDeadline must affect pending socket I/O, not merely record local timestamps.
type ReadLimits struct {
	Idle, Active time.Duration
	SetDeadline  func(time.Time) error
}

// ReadRecord returns exactly one authenticated plaintext envelope. EPN2 binds
// this envelope to exactly one complete EPS2 record. TCP fragmentation remains
// arbitrary; splitting an EPS2 record across AEAD envelopes is not permitted.
// dst must have room for MaxPlaintext; reject before consuming any bytes.
// Do not mix with a partially consumed Read stream. Any error poisons this
// instance: close the physical transport, never retry at a guessed boundary.
func (s *RecordStream) ReadRecord(dst []byte, limits ReadLimits) (n int, err error) {
	if s == nil || s.stream == nil || s.session == nil {
		return 0, ErrConfig
	}
	defer func() {
		if err != nil {
			s.stream = nil
			clear(s.plain[:])
		}
	}()
	if !s.recordReady(limits) {
		return 0, ErrConfig
	}
	if len(dst) < MaxPlaintext {
		return 0, ErrBounds
	}
	n, err = s.readEnvelope(limits)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, ErrBounds
	}
	copy(dst, s.plain[:n])
	s.start, s.end = 0, 0
	return n, nil
}

func (s *RecordStream) recordReady(limits ReadLimits) bool {
	return s.start == s.end && limits.SetDeadline != nil && limits.Active > 0 && limits.Idle >= limits.Active
}

func (s *RecordStream) readEnvelope(limits ReadLimits) (int, error) {
	if err := s.readHeader(limits); err != nil {
		return 0, err
	}
	plainLength := int(binary.BigEndian.Uint16(s.envelope[8:10]))
	if plainLength > MaxPlaintext {
		return 0, ErrBounds
	}
	total := HeaderSize + plainLength + TagSize
	if _, err := io.ReadFull(recordReader{s.stream}, s.envelope[HeaderSize:total]); err != nil {
		return 0, err
	}
	return s.session.Open(s.plain[:], s.envelope[:total])
}

func (s *RecordStream) readHeader(limits ReadLimits) error {
	start := 0
	if limits.SetDeadline != nil {
		if err := limits.SetDeadline(time.Now().Add(limits.Idle)); err != nil {
			return err
		}
		if _, err := io.ReadFull(recordReader{s.stream}, s.envelope[:1]); err != nil {
			return err
		}
		if err := limits.SetDeadline(time.Now().Add(limits.Active)); err != nil {
			return err
		}
		start = 1
	}
	_, err := io.ReadFull(recordReader{s.stream}, s.envelope[start:HeaderSize])
	return err
}

type recordReader struct{ io.Reader }

func (r recordReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if n < 0 || n > len(p) || (n == 0 && err == nil && len(p) != 0) {
		return 0, io.ErrNoProgress
	}
	return n, err
}
