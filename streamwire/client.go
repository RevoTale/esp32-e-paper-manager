package streamwire

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

// Send borrows an immutable server-side frame. The stream must impose I/O
// deadlines. A failure is not automatically retried: completion may be unknown.
// Epoch is a fresh nonzero host random value for this physical USB session.
func Send(stream io.ReadWriter, epoch uint64, frame display.Frame) error {
	if stream == nil || epoch == 0 || frame.Stride() != (frame.Size().Width+7)/8 {
		return ErrRecord
	}
	if frame.Size().Width <= 0 || frame.Size().Height <= 0 {
		return ErrRecord
	}
	c := client{stream: stream, epoch: epoch}
	chunk, err := c.negotiate(frame.Size())
	if err != nil {
		return err
	}
	return c.sendFrame(frame, chunk)
}

func (c *client) negotiate(size display.Size) (int, error) {
	reply, err := c.exchange(Record{Kind: Hello, Epoch: c.epoch})
	if err != nil {
		return 0, err
	}
	if int(binary.LittleEndian.Uint16(reply[8:10])) != size.Width || int(binary.LittleEndian.Uint16(reply[10:12])) != size.Height {
		return 0, ErrRecord
	}
	chunk := int(binary.LittleEndian.Uint16(reply[12:14]))
	if chunk < 1 || chunk > MaxPayload || reply[14] != 2 {
		return 0, ErrRecord
	}
	return chunk, nil
}

func (c *client) sendFrame(frame display.Frame, chunk int) error {
	digest := sha256.Sum256(frame.Bytes())
	if _, err := c.exchange(Record{Kind: Begin, Epoch: c.epoch, ID: 1, Payload: digest[:]}); err != nil {
		return err
	}
	for pass := uint8(0); pass < 2; pass++ {
		if err := c.pass(frame.Bytes(), pass, chunk); err != nil {
			return err
		}
	}
	reply, err := c.exchange(Record{Kind: Commit, Epoch: c.epoch, ID: 1})
	if err != nil {
		return err
	}
	if streamrx.State(reply[1]) != streamrx.Complete {
		return ErrRecord
	}
	return nil
}

type client struct {
	stream io.ReadWriter
	epoch  uint64
	buffer [MaxRecord]byte
}

func (c *client) pass(pixels []byte, pass uint8, chunk int) error {
	for offset := 0; offset < len(pixels); offset += chunk {
		end := min(offset+chunk, len(pixels))
		reply, err := c.exchange(Record{Kind: Data, Epoch: c.epoch, ID: 1, Pass: pass, Offset: uint32(offset), Payload: pixels[offset:end]})
		if err != nil {
			return err
		}
		wantPass, wantOffset := pass, uint32(end)
		if end == len(pixels) {
			wantPass++
			wantOffset = 0
		}
		if reply[2] != wantPass || binary.LittleEndian.Uint32(reply[4:8]) != wantOffset {
			return ErrRecord
		}
	}
	return nil
}

func (c *client) exchange(request Record) ([]byte, error) {
	n, err := Encode(c.buffer[:], request)
	if err != nil {
		return nil, err
	}
	if err = writeRecord(c.stream, c.buffer[:n]); err != nil {
		return nil, err
	}
	if _, err = io.ReadFull(progressReader{c.stream}, c.buffer[:HeaderSize]); err != nil {
		return nil, err
	}
	n, err = Size(c.buffer[:HeaderSize])
	if err != nil {
		return nil, err
	}
	if _, err = io.ReadFull(progressReader{c.stream}, c.buffer[HeaderSize:n]); err != nil {
		return nil, err
	}
	r, err := Decode(c.buffer[:n])
	if err != nil {
		return nil, err
	}
	return validateReply(r, request)
}

func validateReply(r, request Record) ([]byte, error) {
	if r.Kind != Reply || r.Epoch != request.Epoch || r.ID != request.ID || len(r.Payload) != 20 {
		return nil, ErrRecord
	}
	if r.Payload[0] != 0 {
		return nil, fmt.Errorf("stream rejected: operation=%d pass=%d offset=%d code=%d state=%d", request.Kind, request.Pass, request.Offset, r.Payload[0], r.Payload[1])
	}
	return r.Payload, nil
}

type progressReader struct{ io.Reader }

func (r progressReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if n == 0 && err == nil && len(p) > 0 {
		return 0, io.ErrNoProgress
	}
	return n, err
}

func writeRecord(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
		if err != nil {
			return err
		}
		if n <= 0 || n > len(p) {
			return io.ErrShortWrite
		}
		p = p[n:]
	}
	return nil
}
