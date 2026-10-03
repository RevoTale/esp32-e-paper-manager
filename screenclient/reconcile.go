package screenclient

import (
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
	"github.com/RevoTale/esp32-e-paper-manager/streamsession"
)

type Reconciliation struct {
	Confirmed bool
	Status    screenwire.Status
}

// Reconcile never refreshes. Confirmed proves the pending physical operation;
// false requires baseline invalidation and a new complete Send at a new ID.
// Receiving/Ready staging is explicitly aborted before releasing the identity.
func (c *Client) Reconcile() (Reconciliation, error) {
	if !c.bound {
		return Reconciliation{}, ErrConnection
	}
	if !c.Pending() {
		return Reconciliation{}, ErrPending
	}
	r, err := c.exchange(screenwire.Record{Kind: screenwire.Query, Epoch: c.lease.Generation, ID: c.pending.ID, Payload: c.pending.Digest[:]})
	if err != nil {
		return Reconciliation{}, err
	}
	s := r.Status
	if err = c.boundStatus(s); err != nil {
		return Reconciliation{}, err
	}
	if !reconcilable(s.Code) {
		return Reconciliation{}, RemoteError{Status: s}
	}
	result := Reconciliation{Status: s}
	if s.Code == screenwire.CodeOK {
		result.Confirmed = c.complete(s)
	}
	if s.State == streamrx.Receiving || s.State == streamrx.Ready {
		if err = c.abort(); err != nil {
			return Reconciliation{}, err
		}
	}
	c.resolve(result.Confirmed)
	return result, nil
}

func (c *Client) resolve(confirmed bool) {
	c.baseline = [32]byte{}
	if confirmed {
		c.baseline = c.pending.Digest
	}
	c.pending = streamsession.Transaction{}
	c.pendingBytes = 0
}

func (c *Client) boundStatus(s screenwire.Status) error {
	if s.Boot != c.lease.Boot || s.Generation != c.lease.Generation {
		c.baseline = [32]byte{}
		return ErrResync
	}
	if !c.validProgress(s) {
		return screenwire.ErrRecord
	}
	return nil
}

func (c *Client) validProgress(s screenwire.Status) bool {
	limit := uint32(c.caps.Stride) * uint32(c.caps.Height)
	if c.Pending() {
		limit = c.pendingBytes
	}
	if s.Pass > c.caps.Passes || s.Offset >= limit {
		return false
	}
	if s.State == streamrx.Ready || s.State == streamrx.Complete {
		return s.Pass == c.caps.Passes && s.Offset == 0
	}
	if s.Pass == c.caps.Passes {
		return s.Offset == 0 && s.State != streamrx.Receiving
	}
	return true
}

func reconcilable(code screenwire.Code) bool {
	switch code {
	case screenwire.CodeOK, screenwire.CodeStale, screenwire.CodeHardware, screenwire.CodeTimeout,
		screenwire.CodeChunk, screenwire.CodeDigest, screenwire.CodeState:
		return true
	default:
		return false
	}
}

func (c *Client) abort() error {
	r, err := c.exchange(screenwire.Record{Kind: screenwire.Abort, Epoch: c.lease.Generation})
	if err != nil {
		return err
	}
	if err = c.accept(r.Status); err != nil {
		return err
	}
	if r.Status.State != streamrx.Closed {
		return screenwire.ErrRecord
	}
	return nil
}
