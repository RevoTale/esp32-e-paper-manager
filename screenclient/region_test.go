package screenclient

import (
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestRegionRequiresConfirmedBaselineAndCapability(t *testing.T) {
	c, _, _, _, frame := fixture(t)
	old, next := []byte{0x80, 0x80}, []byte{0, 0}
	r := screenwire.RegionBegin{Left: 0, Top: 0, Right: 8, Bottom: 2,
		Old: sha256.Sum256(old), New: sha256.Sum256(next),
		Policy: refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}}
	if err := c.Send(frame); err != nil {
		t.Fatal(err)
	}
	r.Baseline = c.Baseline()
	if r.Baseline != sha256.Sum256(frame.Bytes()) {
		t.Fatal("missing full baseline")
	}
	if err := c.SendRegion(r, old, next); !errors.Is(err, ErrUnsupportedRefresh) {
		t.Fatal(err)
	}
	c.caps.Features |= screenwire.FeatureRegion
	for _, change := range []func(*screenwire.RegionBegin){
		func(r *screenwire.RegionBegin) { r.Baseline[0] ^= 1 },
		func(r *screenwire.RegionBegin) { r.Old[0] ^= 1 },
		func(r *screenwire.RegionBegin) { r.New[0] ^= 1 },
		func(r *screenwire.RegionBegin) { r.Right = 24 },
		func(r *screenwire.RegionBegin) { r.Bottom = 3 },
		func(r *screenwire.RegionBegin) { r.Policy.Normal = 0 },
	} {
		bad := r
		change(&bad)
		if err := c.SendRegion(bad, old, next); err == nil || c.Pending() {
			t.Fatal("invalid region admitted", err)
		}
	}
	if c.Baseline() != r.Baseline {
		t.Fatal("validation destroyed confirmed base")
	}
}

func TestRegionQueryRejectsOffsetOutsideRegion(t *testing.T) {
	c, p, r, old, next := regionFixture(t)
	p.bad = screenwire.Data
	if err := c.SendRegion(r, old, next); err == nil {
		t.Fatal("missing failure")
	}
	p.status.State, p.status.Pass, p.status.Offset = streamrx.Receiving, 0, 2
	if err := c.boundStatus(p.status); !errors.Is(err, screenwire.ErrRecord) || !c.Pending() {
		t.Fatal("out-of-region progress must not release pending identity", err)
	}
}
