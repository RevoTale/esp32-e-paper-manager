package screenclient

import (
	"errors"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

var ErrUnsupportedRefresh = errors.New("screen client: requested refresh policy unsupported")

// SendWithOptions requires explicit peer advertisement. No fallback can turn an
// urgent or partial request into a different operation without the caller knowing.
func (c *Client) SendWithOptions(frame display.Frame, options refreshpolicy.Options, policy refreshpolicy.Policy) error {
	if c.caps.Features&screenwire.FeatureRefreshPolicy == 0 || options.Mode == refreshpolicy.Partial {
		return ErrUnsupportedRefresh
	}
	b, err := screenwire.EncodeRefreshBegin([32]byte{}, options, policy)
	if err != nil {
		return err
	}
	if err := c.prepare(frame); err != nil {
		return err
	}
	copy(b[:32], c.pending.Digest[:])
	r, err := c.exchange(screenwire.Record{Kind: screenwire.BeginRefresh, Epoch: c.lease.Generation, ID: c.pending.ID, Payload: b[:]})
	if err != nil {
		return err
	}
	if err := c.accept(r.Status); err != nil {
		return err
	}
	if r.Status.State != streamrx.Receiving || r.Status.Pass != 0 || r.Status.Offset != 0 {
		return screenwire.ErrRecord
	}
	return c.finish(frame)
}
