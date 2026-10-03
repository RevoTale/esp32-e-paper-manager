package screenclient

import (
	"io"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

// ReadHealth performs exactly one read-only exchange: no Hello, Acquire, Bind,
// transaction or retry. The caller serializes and bounds the borrowed stream;
// network input must already be authenticated. It does not alter another Client.
func ReadHealth(stream io.ReadWriter) (screenwire.Response, error) {
	c := Client{stream: stream}
	r, err := c.exchange(screenwire.Record{Kind: screenwire.Health})
	if err == nil {
		err = remoteOK(r.Status)
	}
	return r, err
}
