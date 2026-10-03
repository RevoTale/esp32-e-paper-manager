package screendelivery

import (
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

// ConfigurePartial sets operator budgets before delivery starts. It does not
// establish peer capability or panel safety, and creates no early permission.
func (c *Cadence) ConfigurePartial(p refreshpolicy.Policy) error {
	if _, err := screenwire.EncodeRefreshBegin([32]byte{}, refreshpolicy.Options{}, p); err != nil {
		return err
	}
	c.PartialPolicy = p
	return nil
}

func (c *Cadence) completePartial(now time.Time) {
	if c.PartialPolicy.Normal > 0 {
		c.partial = now.Add(c.PartialPolicy.Normal)
		c.partialUrgent = now.Add(c.PartialPolicy.Urgent)
	}
}

// An ambiguous physical completion cannot inherit the fast confirmed lane.
// Preserve longer existing guards and operator budgets independently per lane.
func (c *Cadence) unknownPartial(now, floor time.Time) {
	if c.PartialPolicy.Normal > 0 {
		c.partial = latestDeadline(c.partial, floor, now.Add(c.PartialPolicy.Normal))
		c.partialUrgent = latestDeadline(c.partialUrgent, floor, now.Add(c.PartialPolicy.Urgent))
	}
}

func latestDeadline(a, b, d time.Time) time.Time {
	if b.After(a) {
		a = b
	}
	if d.After(a) {
		a = d
	}
	return a
}

// Reset preserves configured policy but revokes every confirmed deadline.
// The transport must establish a conservative Unknown guard on reconnection.
func (c *Cadence) Reset() {
	c.Urgent, c.partial, c.partialUrgent = time.Time{}, time.Time{}, time.Time{}
}
