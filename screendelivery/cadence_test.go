package screendelivery

import (
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
)

func TestCadenceUnknownVersusConfirmed(t *testing.T) {
	var c Cadence
	now := time.Unix(10, 0)
	urgent := refreshpolicy.Options{Priority: refreshpolicy.Urgent}
	if c.Enabled() || c.Deadline(now, urgent) != now {
		t.Fatal(c)
	}
	if c.Ready(now).NotBefore != now {
		t.Fatal(c)
	}
	if c.Configure(refreshpolicy.Policy{}) == nil {
		t.Fatal("invalid")
	}
	p := refreshpolicy.Policy{Normal: time.Minute, Urgent: time.Second}
	if err := c.Configure(p); err != nil {
		t.Fatal(err)
	}
	if c.Ready(now).UrgentNotBefore != now || c.Deadline(now, urgent) != now {
		t.Fatal("unknown bypass")
	}
	checkCompletedCadence(t, &c, now, urgent)
}

func TestUnknownCadenceHonorsLongerOperatorIntervals(t *testing.T) {
	c := Cadence{Policy: refreshpolicy.Policy{Normal: time.Minute, Urgent: 2 * time.Minute}}
	now := time.Unix(10, 0)
	floor := c.Unknown(now, time.Time{}, time.Second)
	if floor != now.Add(time.Minute) || c.Urgent != now.Add(2*time.Minute) {
		t.Fatal(floor, c)
	}
	later := now.Add(3 * time.Minute)
	if c.Unknown(now, later, time.Second) != later || c.Urgent != later {
		t.Fatal(c)
	}
}

func checkCompletedCadence(t *testing.T, c *Cadence, now time.Time, urgent refreshpolicy.Options) {
	t.Helper()
	normal := c.Complete(now)
	if normal != now.Add(time.Minute) || c.Deadline(normal, urgent) != now.Add(time.Second) || c.Deadline(normal, refreshpolicy.Options{}) != normal {
		t.Fatal(c)
	}
	if c.Ready(normal).UrgentNotBefore != now.Add(time.Second) {
		t.Fatal(c)
	}
}
