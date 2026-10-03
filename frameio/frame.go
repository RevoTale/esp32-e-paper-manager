// Package frameio defines the bounded BZM1 local process-pipe frame format.
// It is not a device/network protocol and carries no authentication or revision.
package frameio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

var ErrFrame = errors.New("frameio: invalid packed frame")

// Decode borrows a complete in-memory BZM1 record, avoiding a second copy of
// already bounded worker output. Callers retain the backing bytes unchanged.
func Decode(data []byte) (display.Frame, error) {
	if len(data) < 8 {
		return display.Frame{}, ErrFrame
	}
	size := display.Size{Width: int(binary.LittleEndian.Uint16(data[4:6])), Height: int(binary.LittleEndian.Uint16(data[6:8]))}
	if string(data[:4]) != "BZM1" || !validSize(size) {
		return display.Frame{}, ErrFrame
	}
	f, err := display.NewFrame(size, (size.Width+7)/8, data[8:])
	if err != nil || !valid(f) {
		return display.Frame{}, ErrFrame
	}
	return f, nil
}

// Reader borrows immutable frame storage until fully consumed; no pixel copy.
func Reader(frame display.Frame) (io.Reader, error) {
	if !valid(frame) {
		return nil, ErrFrame
	}
	var header [8]byte
	copy(header[:], "BZM1")
	binary.LittleEndian.PutUint16(header[4:6], uint16(frame.Size().Width))
	binary.LittleEndian.PutUint16(header[6:8], uint16(frame.Size().Height))
	return io.MultiReader(bytes.NewReader(header[:]), bytes.NewReader(frame.Bytes())), nil
}

// Read owns returned pixels. Validate geometry before allocating and require
// EOF after one complete frame. The pipe owner must enforce a process deadline.
func Read(r io.Reader) (display.Frame, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return display.Frame{}, err
	}
	size := display.Size{Width: int(binary.LittleEndian.Uint16(header[4:6])), Height: int(binary.LittleEndian.Uint16(header[6:8]))}
	if string(header[:4]) != "BZM1" || !validSize(size) {
		return display.Frame{}, ErrFrame
	}
	stride := (size.Width + 7) / 8
	pixels, err := io.ReadAll(io.LimitReader(r, int64(stride*size.Height+1)))
	if err != nil {
		return display.Frame{}, err
	}
	frame, err := display.NewFrame(size, stride, pixels)
	if err != nil || !valid(frame) {
		return display.Frame{}, ErrFrame
	}
	return frame, nil
}

func validSize(s display.Size) bool {
	return s.Width > 0 && s.Height > 0 && s.Width <= 2048 && s.Height <= 2048 && s.Width*s.Height <= 1_048_576
}

func valid(f display.Frame) bool {
	if !validSize(f.Size()) || f.Stride() != (f.Size().Width+7)/8 {
		return false
	}
	if f.Size().Width%8 == 0 {
		return true
	}
	mask := byte(0xff >> uint(f.Size().Width%8))
	for y := 0; y < f.Size().Height; y++ {
		if f.Bytes()[(y+1)*f.Stride()-1]&mask != 0 {
			return false
		}
	}
	return true
}
