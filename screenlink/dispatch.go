package screenlink

import (
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
	"github.com/RevoTale/esp32-e-paper-manager/streamsession"
)

func (c *Connection) dispatch(r screenwire.Record, now time.Duration) (streamsession.Result, error) {
	if r.Kind == screenwire.PanelTrace {
		if c.device.panelTrace == nil {
			return streamsession.Result{}, streamrx.ErrConfig
		}
		return streamsession.Result{}, nil // Cached observation must not invoke Tick.
	}
	if r.Kind == screenwire.Health {
		// Tick can abort expired staging through the panel sink. A read-only
		// health query must not become the trigger for that physical operation.
		if c.device.health == nil {
			return streamsession.Result{}, streamrx.ErrConfig
		}
		return streamsession.Result{}, nil
	}
	if err := c.device.Tick(now); err != nil {
		return c.snapshot(c.tx), err
	}
	switch r.Kind {
	case screenwire.Hello:
		return streamsession.Result{}, nil
	case screenwire.Acquire:
		return streamsession.Result{}, c.acquire(r)
	case screenwire.Bind:
		return streamsession.Result{}, c.bind(r)
	default:
		return c.apply(r, now)
	}
}

func (c *Connection) acquire(r screenwire.Record) error {
	var boot, claim [16]byte
	copy(boot[:], r.Payload[:16])
	copy(claim[:], r.Payload[16:])
	_, err := c.device.session.Acquire(streamsession.Epoch{Boot: boot, Generation: r.Epoch}, claim)
	return err
}

func (c *Connection) bind(r screenwire.Record) error {
	lease := streamsession.Lease{Epoch: streamsession.Epoch{Generation: r.Epoch}}
	copy(lease.Boot[:], r.Payload[:16])
	copy(lease.Claim[:], r.Payload[16:])
	if c.binding.Serial != 0 {
		if c.binding.Lease == lease {
			return nil
		}
		return streamsession.ErrLease
	}
	binding, err := c.device.session.Bind(lease)
	if err == nil {
		c.binding = binding
		c.device.active = c
	}
	return err
}

func (c *Connection) apply(r screenwire.Record, now time.Duration) (streamsession.Result, error) {
	if c.binding.Serial == 0 || r.Epoch != c.binding.Generation {
		return streamsession.Result{}, streamsession.ErrLease
	}
	switch r.Kind {
	case screenwire.Abort:
		return c.abort()
	case screenwire.Data, screenwire.DataPacked:
		return c.data(r, now)
	}
	tx := streamsession.Transaction{Lease: c.binding.Lease, ID: r.ID}
	copy(tx.Digest[:], r.Payload)
	s := c.device.session
	var err error
	switch r.Kind {
	case screenwire.Begin:
		err = s.Begin(c.binding, tx, now)
		if err == nil {
			c.tx = tx
		}
	case screenwire.Commit:
		err = s.Commit(c.binding, tx, now)
	case screenwire.Query:
		return s.Query(c.binding, tx)
	default:
		err = screenwire.ErrRecord
	}
	return c.snapshot(tx), err
}

func (c *Connection) data(r screenwire.Record, now time.Duration) (streamsession.Result, error) {
	if r.ID != c.tx.ID || c.tx.ID == 0 {
		return streamsession.Result{}, streamrx.ErrState
	}
	if r.Kind == screenwire.DataPacked {
		pixels, err := c.unpack(r.Payload)
		if err != nil {
			return c.rejectPacked(err)
		}
		r.Payload = pixels
	}
	err := c.device.session.Write(c.binding, c.tx, r.Pass, r.Offset, r.Payload, now)
	return c.snapshot(c.tx), err
}

func (c *Connection) abort() (streamsession.Result, error) {
	err := c.device.session.Abort(c.binding)
	return c.snapshot(c.tx), err
}

func (c *Connection) snapshot(tx streamsession.Transaction) streamsession.Result {
	result, _ := c.device.session.Query(c.binding, tx)
	return result
}
