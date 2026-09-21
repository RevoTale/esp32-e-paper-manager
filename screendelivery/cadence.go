package screendelivery

import (
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

// Cadence belongs to the single transport owner. Unknown completion receives
// the normal interval even for urgent callers; only a known ACK opens that lane.
type Cadence struct {
	Policy refreshpolicy.Policy
	Urgent time.Time
}

func (c *Cadence) Configure(p refreshpolicy.Policy) error {
	if _, err := screenwire.EncodeRefreshBegin([32]byte{}, refreshpolicy.Options{}, p); err != nil {
		return err
	}
	c.Policy = p
	return nil
}

func (c *Cadence) Enabled() bool { return c.Policy.Normal > 0 }

func (c *Cadence) Complete(now time.Time) time.Time {
	c.Urgent = now.Add(c.Policy.Urgent)
	return now.Add(c.Policy.Normal)
}

// Unknown retains the legacy reboot/reconciliation guard and any longer local
// policy. An urgent interval longer than normal must still be honored.
func (c *Cadence) Unknown(now, existing time.Time, legacy time.Duration) time.Time {
	normal := max(legacy, c.Policy.Normal)
	floor := now.Add(normal)
	if existing.After(floor) {
		floor = existing
	}
	c.Urgent = now.Add(max(normal, c.Policy.Urgent))
	if floor.After(c.Urgent) {
		c.Urgent = floor
	}
	return floor
}

func (c *Cadence) Ready(normal time.Time) Readiness {
	urgent := c.Urgent
	if !c.Enabled() || urgent.IsZero() {
		urgent = normal
	}
	return Readiness{NotBefore: normal, UrgentNotBefore: urgent}
}

func (c *Cadence) Deadline(normal time.Time, o refreshpolicy.Options) time.Time {
	if c.Enabled() && o.Priority == refreshpolicy.Urgent && !c.Urgent.IsZero() {
		return c.Urgent
	}
	return normal
}
