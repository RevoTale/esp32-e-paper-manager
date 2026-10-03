package panel

import (
	"errors"
	"sync/atomic"
)

var ErrStreamState = errors.New("panel: stream state or bounds")

// Stream is an experimental single-owner two-pass full-refresh sink. The caller
// verifies both passes before Commit and enforces refresh cadence. It must Abort
// on invalid input, disconnect or idle timeout. Only the 100-byte inversion row
// is retained. No partial writes/refreshes or plane-pointer resumption is assumed.
// Sequence source: https://github.com/waveshareteam/Pico_ePaper_Code/blob/c9bcd84db5adf5f085353649a8a5c31492bc5fb8/c/lib/e-Paper/EPD_7in5_V2.c
type Stream struct {
	d      *Driver
	active bool
	pass   uint8
	offset uint32
}

func NewStream(driver *Driver) (*Stream, error) {
	if driver == nil {
		return nil, ErrConfig
	}
	return &Stream{d: driver}, nil
}

func (s *Stream) Begin() error {
	if s.active || !atomic.CompareAndSwapUint32(&s.d.active, 0, 1) {
		return ErrInUse
	}
	s.active = true
	s.pass, s.offset = 0, 0
	if err := s.d.start(); err != nil {
		return errors.Join(err, s.Abort())
	}
	return nil
}

func (s *Stream) Write(pass uint8, offset uint32, pixels []byte) error {
	if !s.active || pass != s.pass || offset != s.offset {
		return ErrStreamState
	}
	if !validStreamChunk(pass, offset, len(pixels)) {
		return ErrStreamState
	}
	phase, step, command := PhaseOldPlane, StepOldPlane, byte(0x10)
	if pass == 1 {
		phase, step, command = PhaseNewPlane, StepNewPlane, 0x13
	}
	if offset == 0 {
		if err := s.d.sendCommand(phase, step, command); err != nil {
			return err
		}
	}
	return s.writeData(phase, step, command, pixels)
}

func validStreamChunk(pass uint8, offset uint32, size int) bool {
	return pass < 2 && size > 0 && size <= rowBytes && uint64(offset)+uint64(size) <= FrameBytes
}

func (s *Stream) writeData(phase Phase, step Step, command byte, pixels []byte) error {
	data := pixels
	if s.pass == 0 {
		for i, value := range pixels {
			s.d.row[i] = ^value
		}
		data = s.d.row[:len(pixels)]
	}
	if err := s.d.data(data); err != nil {
		return OpError{Phase: phase, Step: step, Command: command, Offset: int(s.offset), Cause: err}
	}
	s.offset += uint32(len(pixels))
	if s.offset == FrameBytes {
		s.offset = 0
		s.pass++
	}
	return nil
}

func (s *Stream) Commit() error {
	if !s.active || s.pass != 2 {
		return ErrStreamState
	}
	defer func() { _ = s.Abort() }()
	return s.d.finish()
}

func (s *Stream) Abort() error {
	if s.active {
		s.d.releasePower()
		s.active = false
		atomic.StoreUint32(&s.d.active, 0)
	}
	return nil
}
