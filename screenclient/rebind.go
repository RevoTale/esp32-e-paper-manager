package screenclient

// Rebind renews only a remembered lease on a still-open, idle USB-UART stream.
// Unlike read-only Probe it sends Bind, but never Acquire, pixels or Refresh.
// Use after an adapter's logical idle disconnect, with no concurrent I/O. An
// unknown update must use Connect/Reconcile instead; it is never replayed here.
func (c *Client) Rebind() error {
	if c.Pending() {
		return ErrPending
	}
	if c.stream == nil || c.lease.Generation == 0 {
		return ErrConnection
	}
	_, err := c.Connect(c.stream)
	return err
}
