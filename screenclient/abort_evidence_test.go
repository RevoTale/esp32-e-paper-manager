package screenclient

import (
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestAbortRequiresTerminalEvidence(t *testing.T) {
	c, _, rw, _, frame := fixture(t)
	if err := c.start(frame); err != nil {
		t.Fatal(err)
	}
	rewriteStatus(t, rw, screenwire.Abort, func(s *screenwire.Status) { s.State = streamrx.Receiving; s.Pass = 0; s.Offset = 0 })
	if _, err := c.Reconcile(); !errors.Is(err, screenwire.ErrRecord) || !c.Pending() {
		t.Fatal(err, c.Pending())
	}
}
