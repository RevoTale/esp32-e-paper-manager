package screenclient

import (
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"io"
)

// ReadPanelTrace borrows a serialized, bounded, already-trusted stream for one
// read-only exchange. No retry, lease acquisition, or panel operation occurs.
func ReadPanelTrace(stream io.ReadWriter) (screenwire.Response, error) {
	c := Client{stream: stream}
	r, err := c.exchange(screenwire.Record{Kind: screenwire.PanelTrace})
	if err == nil {
		err = remoteOK(r.Status)
	}
	return r, err
}
