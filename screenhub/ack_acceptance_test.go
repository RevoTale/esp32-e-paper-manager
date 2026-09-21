package screenhub

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"io"
	"net"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

var liveACKEnrollment = flag.String("epaper-live-enrollment", "", "explicit physical ACK-loss test enrollment; default uses simulator")
var liveACKListen = flag.String("epaper-live-listen", "", "explicit physical ACK-loss test listen address")
var liveACKConfirm = flag.Bool("epaper-live-confirm", false, "authorize one physical full frame; operator available for power removal")

// Always runs against the simulator in normal tests. The same scenario can be
// selected explicitly in a native test binary for the enrolled physical panel.
func TestCommitACKAcceptance(t *testing.T) {
	connect, wait, frame, verify := ackEnvironment(t, 1)
	c, err := screenclient.New(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	first := &ackCut{ReadWriteCloser: connect(), armed: true}
	t.Cleanup(func() { _ = first.Close() })
	caps, err := c.Connect(first)
	if err != nil {
		t.Fatal(err)
	}
	checkACKProfile(t, caps, frame)
	wait(time.Duration(caps.MinimumFullMS) * time.Millisecond)
	err = c.Send(frame)
	checkACKLoss(t, first, c, err)
	t.Log("terminal Commit ACK validated, withheld; socket closed; pending client retained")
	second := reconnectACK(t, c, connect)
	t.Cleanup(func() { _ = second.Close() })
	r, err := c.Reconcile()
	if err != nil || !r.Confirmed || c.Pending() {
		t.Fatalf("reconciliation failed: confirmed=%t error=%v", r.Confirmed, err)
	}
	checkACKCounts(t, first, second)
	verify()
	t.Log("reconciled=confirmed commit_requests=1 reconnect_commit_requests=0 queries=1")
}

type ackConnect func() io.ReadWriteCloser

func reconnectACK(t *testing.T, c *screenclient.Client, connect ackConnect) *ackCut {
	t.Helper()
	for attempt := 1; attempt <= 3; attempt++ {
		p := &ackCut{ReadWriteCloser: connect()}
		_, err := c.Connect(p)
		if err == nil {
			return p
		}
		_ = p.Close()
		if !transient(err) {
			t.Fatal(err)
		}
		t.Logf("transient reconnect failure attempt=%d; same client retained; no Send: %v", attempt, err)
	}
	t.Fatal("reconnect budget exhausted; no replay")
	return nil
}

type failedACKStream struct{}

func (failedACKStream) Read([]byte) (int, error)  { return 0, net.ErrClosed }
func (failedACKStream) Write([]byte) (int, error) { return 0, net.ErrClosed }
func (failedACKStream) Close() error              { return nil }

func checkACKProfile(t *testing.T, caps screenwire.Capabilities, frame display.Frame) {
	t.Helper()
	if caps.Width != uint16(frame.Size().Width) || caps.Height != uint16(frame.Size().Height) || caps.Profile != 1 || caps.ProfileVersion != 1 {
		t.Fatal("profile mismatch; no frame sent")
	}
}

func checkACKLoss(t *testing.T, p *ackCut, c *screenclient.Client, err error) {
	t.Helper()
	if !p.cut || !errors.Is(err, io.ErrUnexpectedEOF) || !c.Pending() {
		t.Fatalf("expected injected ACK loss: cut=%t pending=%t error=%v", p.cut, c.Pending(), err)
	}
}

func checkACKCounts(t *testing.T, first, second *ackCut) {
	t.Helper()
	if first.commits != 1 || second.commits != 0 || second.queries != 1 {
		t.Fatal("unexpected refresh replay or query count")
	}
}

func ackEnvironment(t *testing.T, commits int32) (ackConnect, func(time.Duration), display.Frame, func()) {
	t.Helper()
	if *liveACKEnrollment != "" || *liveACKListen != "" || *liveACKConfirm {
		return liveACKEnvironment(t)
	}
	f := deviceTest(t, 1)
	calls := 0
	connect := func() io.ReadWriteCloser {
		calls++
		if calls == 2 {
			return failedACKStream{}
		}
		return simulatedACKStream(t, f)
	}
	return connect, func(time.Duration) {}, f.frame, func() {
		if f.sink.commits.Load() != commits {
			t.Fatalf("physical simulator commits=%d want=%d", f.sink.commits.Load(), commits)
		}
	}
}

func boundedACKContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	t.Cleanup(cancel)
	return ctx
}
