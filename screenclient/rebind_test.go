package screenclient

import (
	"errors"
	"reflect"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestRebindUsesExistingClaimWithoutRefreshing(t *testing.T) {
	c, _, rw, sink, frame := fixture(t)
	lease := c.lease
	var kinds []screenwire.Kind
	rw.rewrite = func(request screenwire.Record, reply []byte) []byte {
		kinds = append(kinds, request.Kind)
		return reply
	}
	if err := c.Rebind(); err != nil {
		t.Fatal(err)
	}
	if c.lease != lease || sink.commits != 0 || !reflect.DeepEqual(kinds, []screenwire.Kind{screenwire.Hello, screenwire.Bind}) {
		t.Fatal("rebind acquired, refreshed or changed ownership", kinds)
	}
	if err := c.Send(frame); err != nil {
		t.Fatal(err)
	}
}

func TestRebindRejectsPendingAndChangedLease(t *testing.T) {
	c, _, rw, _, frame := fixture(t)
	rw.loseCommit = true
	if err := c.Send(frame); err == nil {
		t.Fatal("expected missing ACK")
	}
	if err := c.Rebind(); !errors.Is(err, ErrPending) {
		t.Fatal(err)
	}
	c, _, _, _, _ = fixture(t)
	c.lease.Generation++
	if err := c.Rebind(); !errors.Is(err, ErrResync) {
		t.Fatal(err)
	}
	var empty Client
	if err := empty.Rebind(); !errors.Is(err, ErrConnection) {
		t.Fatal(err)
	}
}
