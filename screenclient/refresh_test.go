package screenclient

import (
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestRefreshRejectsBeforeConsumingIdentity(t *testing.T) {
	c, _, rw, sink, f := fixture(t)
	p := refreshpolicy.Policy{Normal: time.Minute, Urgent: time.Second}
	before := rw.wireBytes
	if err := c.SendWithOptions(f, refreshpolicy.Options{Priority: refreshpolicy.Urgent}, p); !errors.Is(err, ErrUnsupportedRefresh) {
		t.Fatal(err)
	}
	c.caps.Features |= screenwire.FeatureRefreshPolicy
	if err := c.SendWithOptions(f, refreshpolicy.Options{Mode: refreshpolicy.Partial}, p); !errors.Is(err, ErrUnsupportedRefresh) {
		t.Fatal(err)
	}
	if err := c.SendWithOptions(f, refreshpolicy.Options{}, refreshpolicy.Policy{}); !errors.Is(err, screenwire.ErrRecord) {
		t.Fatal(err)
	}
	if c.Pending() || c.next != 0 || rw.wireBytes != before || sink.commits != 0 {
		t.Fatal("rejected options mutated transaction")
	}
}
