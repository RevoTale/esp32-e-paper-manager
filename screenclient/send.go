package screenclient

import (
	"crypto/sha256"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
	"github.com/RevoTale/esp32-e-paper-manager/streamsession"
)

// Send borrows immutable pixels until return. A failure retains its identity
// but no pixel slice; Reconcile must resolve it before another Send is allowed.
func (c *Client) Send(frame display.Frame) error {
	if err := c.start(frame); err != nil {
		return err
	}
	return c.finish(frame)
}

func (c *Client) finish(frame display.Frame) error {
	for pass := uint8(0); pass < c.caps.Passes; pass++ {
		if err := c.sendPass(frame.Bytes(), pass); err != nil {
			return err
		}
	}
	r, err := c.transaction(screenwire.Commit)
	if err != nil {
		return err
	}
	if !c.complete(r) {
		return screenwire.ErrRecord
	}
	c.pending = streamsession.Transaction{}
	return nil
}

func (c *Client) prepare(frame display.Frame) error {
	if !c.bound {
		return ErrConnection
	}
	if c.Pending() {
		return ErrPending
	}
	if !c.validFrame(frame) || c.next == ^uint64(0) {
		return screenwire.ErrRecord
	}
	c.next++
	c.pending = streamsession.Transaction{Lease: c.lease, ID: c.next, Digest: sha256.Sum256(frame.Bytes())}
	return nil
}

func (c *Client) start(frame display.Frame) error {
	if err := c.prepare(frame); err != nil {
		return err
	}
	r, err := c.transaction(screenwire.Begin)
	if err != nil {
		return err
	}
	if r.State != streamrx.Receiving || r.Pass != 0 || r.Offset != 0 {
		return screenwire.ErrRecord
	}
	return nil
}

func (c *Client) validFrame(f display.Frame) bool {
	if f.Size() != (display.Size{Width: int(c.caps.Width), Height: int(c.caps.Height)}) || f.Stride() != int(c.caps.Stride) {
		return false
	}
	if len(f.Bytes()) != int(c.caps.Stride)*int(c.caps.Height) {
		return false
	}
	if c.caps.Width%8 == 0 {
		return true
	}
	mask := byte(0xff >> (c.caps.Width % 8))
	for i := f.Stride() - 1; i < len(f.Bytes()); i += f.Stride() {
		if f.Bytes()[i]&mask != 0 {
			return false
		}
	}
	return true
}

func (c *Client) transaction(kind screenwire.Kind) (screenwire.Status, error) {
	r, err := c.exchange(screenwire.Record{Kind: kind, Epoch: c.lease.Generation, ID: c.pending.ID, Payload: c.pending.Digest[:]})
	if err != nil {
		return screenwire.Status{}, err
	}
	return r.Status, c.accept(r.Status)
}

func (c *Client) sendPass(pixels []byte, pass uint8) error {
	var packed [screenwire.MaxPayload]byte
	for offset := 0; offset < len(pixels); {
		end := min(offset+int(min(c.caps.MaxChunk, c.maxChunk)), len(pixels))
		kind, payload := c.packChunk(packed[:], pixels[offset:end])
		r, err := c.exchange(screenwire.Record{Kind: kind, Epoch: c.lease.Generation, ID: c.pending.ID,
			Pass: pass, Offset: uint32(offset), Payload: payload})
		if err != nil {
			return err
		}
		if err = c.accept(r.Status); err != nil {
			return err
		}
		if !c.progress(r.Status, pass, end, len(pixels)) {
			return screenwire.ErrRecord
		}
		offset = end
	}
	return nil
}

func (c *Client) progress(s screenwire.Status, pass uint8, end, total int) bool {
	nextPass, nextOffset, state := pass, uint32(end), streamrx.Receiving
	if end == total {
		nextPass++
		nextOffset = 0
	}
	if nextPass == c.caps.Passes {
		state = streamrx.Ready
	}
	return s.Pass == nextPass && s.Offset == nextOffset && s.State == state
}

func (c *Client) complete(s screenwire.Status) bool {
	return s.State == streamrx.Complete && s.Pass == c.caps.Passes && s.Offset == 0 && s.CurrentImage
}
