package screenclient

import "github.com/RevoTale/esp32-e-paper-manager/screenwire"

// Probe checks a remembered connection using read-only Hello. It cannot adopt
// ownership, reconcile an unknown update, or perform a physical refresh. The
// single client owner must reconcile pending identity before an idle probe.
func (c *Client) Probe() error {
	if c.Pending() {
		return ErrPending
	}
	if !c.bound {
		return ErrConnection
	}
	r, err := c.exchange(screenwire.Record{Kind: screenwire.Hello})
	if err != nil {
		return err
	}
	if err = remoteOK(r.Status); err != nil {
		return err
	}
	if r.Status.Boot != c.lease.Boot || r.Status.Generation != c.lease.Generation || r.Capabilities != c.caps {
		c.bound = false
		return ErrResync
	}
	return nil
}
