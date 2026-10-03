package screenclient

import (
	"crypto/sha256"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
	"github.com/RevoTale/esp32-e-paper-manager/streamsession"
)

// Baseline is the last confirmed transaction identity, not necessarily a pixel
// hash. Zero means unavailable; the caller owns the corresponding pixel snapshot.
func (c *Client) Baseline() [32]byte {
	if !c.bound || c.Pending() {
		return [32]byte{}
	}
	return c.baseline
}

// SendRegion borrows immutable canonical old/new region pixels until return.
// No refresh is retried. An admitted failure must be resolved by Reconcile.
func (c *Client) SendRegion(region screenwire.RegionBegin, old, next []byte) error {
	b, err := c.regionPayload(region, old, next)
	if err != nil {
		return err
	}
	c.next++
	c.pending = streamsession.Transaction{Lease: c.lease, ID: c.next, Digest: [32]byte(b[:32])}
	c.baseline = [32]byte{}
	c.pendingBytes = uint32(len(old))
	r, err := c.exchange(screenwire.Record{Kind: screenwire.BeginRegion, Epoch: c.lease.Generation, ID: c.next, Payload: b[:]})
	if err != nil {
		return err
	}
	if err := c.accept(r.Status); err != nil {
		return err
	}
	if r.Status.State != streamrx.Receiving || r.Status.Pass != 0 || r.Status.Offset != 0 {
		return screenwire.ErrRecord
	}
	if err := c.sendPass(old, 0); err != nil {
		return err
	}
	if err := c.sendPass(next, 1); err != nil {
		return err
	}
	return c.finishTransaction()
}

func (c *Client) regionPayload(r screenwire.RegionBegin, old, next []byte) ([screenwire.RegionBeginSize]byte, error) {
	if err := c.regionReady(); err != nil {
		return [screenwire.RegionBeginSize]byte{}, err
	}
	b, err := screenwire.EncodeRegionBegin(r)
	if err != nil {
		return b, err
	}
	if _, err = screenwire.DecodeRegionBegin(b[:], c.caps.Width, c.caps.Height); err != nil {
		return b, err
	}
	n := int(r.Right-r.Left) / 8 * int(r.Bottom-r.Top)
	if c.next == ^uint64(0) || r.Baseline != c.baseline || len(old) != n || len(next) != n ||
		sha256.Sum256(old) != r.Old || sha256.Sum256(next) != r.New {
		return b, screenwire.ErrRecord
	}
	return b, nil
}

func (c *Client) regionReady() error {
	if !c.bound {
		return ErrConnection
	}
	if c.Pending() {
		return ErrPending
	}
	if c.caps.Features&screenwire.FeatureRegion == 0 {
		return ErrUnsupportedRefresh
	}
	return nil
}
