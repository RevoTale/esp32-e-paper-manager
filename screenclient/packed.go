package screenclient

import (
	"encoding/binary"

	"github.com/RevoTale/esp32-e-paper-manager/packbits"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

// Negotiation is mandatory. Prefix cost is included, so this cannot enlarge a
// data record or change ACK count. Hashes/offsets remain canonical decoded bytes.
func (c *Client) packChunk(scratch, raw []byte) (screenwire.Kind, []byte) {
	if c.caps.Features&screenwire.FeaturePackBits != 0 {
		n, err := packbits.Encode(scratch[2:], raw)
		if err == nil && n+2 < len(raw) {
			binary.LittleEndian.PutUint16(scratch[:2], uint16(len(raw)))
			return screenwire.DataPacked, scratch[:n+2]
		}
	}
	return screenwire.Data, raw
}
