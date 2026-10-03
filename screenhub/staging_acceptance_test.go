package screenhub

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestStagingInterruptionAcceptance(t *testing.T) {
	connect, wait, frame, verify := ackEnvironment(t, 0)
	if err := frame.Clear(display.Black); err != nil {
		t.Fatal(err)
	}
	c, err := screenclient.New(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	first := &ackCut{ReadWriteCloser: connect(), armed: true, target: screenwire.Data}
	t.Cleanup(func() { _ = first.Close() })
	caps, err := c.Connect(first)
	if err != nil {
		t.Fatal(err)
	}
	checkACKProfile(t, caps, frame)
	wait(time.Duration(caps.MinimumFullMS) * time.Millisecond)
	err = c.Send(frame)
	checkACKLoss(t, first, c, err)
	if first.commits != 0 {
		t.Fatal("staging cut sent Commit")
	}
	t.Log("first Data ACK validated; cut before Commit; pending client retained")
	second := reconnectACK(t, c, connect)
	t.Cleanup(func() { _ = second.Close() })
	r, err := c.Reconcile()
	checkStagingResult(t, r, err, c.Pending())
	if second.commits != 0 || second.queries != 1 {
		t.Fatal("unexpected Commit or Query count")
	}
	verify()
	t.Logf("reconciled=unconfirmed commit_requests=0 reconnect_commit_requests=0 queries=1 code=%d state=%d", r.Status.Code, r.Status.State)
}

func checkStagingResult(t *testing.T, r screenclient.Reconciliation, err error, pending bool) {
	t.Helper()
	if err != nil || r.Confirmed || pending || r.Status.Code == screenwire.CodeHardware {
		t.Fatalf("staging cleanup not accepted: confirmed=%t pending=%t code=%d error=%v", r.Confirmed, pending, r.Status.Code, err)
	}
}
