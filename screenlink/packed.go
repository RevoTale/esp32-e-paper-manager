package screenlink

import (
	"encoding/binary"
	"errors"

	"github.com/RevoTale/esp32-e-paper-manager/packbits"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
	"github.com/RevoTale/esp32-e-paper-manager/streamsession"
)

// Called only after authentication, binding and exact transaction-ID checks.
// Decode the complete bounded block before receiver padding/offset/SHA checks
// and any sink.Write; malformed scratch never reaches controller RAM.
func (c *Connection) unpack(payload []byte) ([]byte, error) {
	if c.device.caps.Features&screenwire.FeaturePackBits == 0 || len(payload) < 4 {
		return nil, streamrx.ErrChunk
	}
	n := int(binary.LittleEndian.Uint16(payload[:2]))
	if n == 0 || n > int(c.device.caps.MaxChunk) {
		return nil, streamrx.ErrChunk
	}
	pixels := c.device.decoded[:n]
	if err := packbits.Decode(pixels, payload[2:]); err != nil {
		return nil, streamrx.ErrChunk
	}
	return pixels, nil
}

func (c *Connection) rejectPacked(cause error) (streamsession.Result, error) {
	// A rejected compressed block must not leave a Ready transaction committable.
	// Preserve cleanup failures in the returned error; do not release the binding.
	err := c.device.session.Abort(c.binding)
	return c.snapshot(c.tx), errors.Join(cause, err)
}
