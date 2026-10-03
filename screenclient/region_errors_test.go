package screenclient

import (
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestRegionAmbiguousBoundaries(t *testing.T) {
	for _, setup := range []func(*regionPeer){
		func(p *regionPeer) { p.lose = screenwire.BeginRegion },
		func(p *regionPeer) { p.bad, p.badPass = screenwire.Data, 1 },
		func(p *regionPeer) { p.mutate = func(s *screenwire.Status) { s.Pass = 1 } },
		func(p *regionPeer) { p.mutate = func(s *screenwire.Status) { s.Offset = 1 } },
		func(p *regionPeer) { p.mutate = func(s *screenwire.Status) { s.State = streamrx.Failed } },
	} {
		c, p, r, old, next := regionFixture(t)
		setup(p)
		if err := c.SendRegion(r, old, next); err == nil || !c.Pending() {
			t.Fatal("ambiguous transaction released", err)
		}
		if p.commits != 0 || c.Baseline() != [32]byte{} {
			t.Fatal("unproven refresh")
		}
	}
}

func TestRegionValidationDoesNotConsumeID(t *testing.T) {
	for _, setup := range []func(*Client, *[]byte, *[]byte){
		func(c *Client, _, _ *[]byte) { c.next = ^uint64(0) },
		func(_ *Client, old, _ *[]byte) { *old = (*old)[:1] },
		func(_ *Client, _, next *[]byte) { *next = (*next)[:1] },
	} {
		c, p, r, old, next := regionFixture(t)
		setup(c, &old, &next)
		id := c.next
		if err := c.SendRegion(r, old, next); err == nil {
			t.Fatal("invalid input")
		}
		if c.Pending() || c.next != id || c.Baseline() != r.Baseline || p.status.Operation != 0 {
			t.Fatal("pre-admission rejection mutated state")
		}
	}
}
