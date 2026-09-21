package screenclient

import (
	"errors"
	"io"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestProbeIsReadOnlyAndRejectsChangedIdentity(t *testing.T) {
	for _, change := range []string{"none", "boot", "generation", "caps"} {
		t.Run(change, func(t *testing.T) {
			c, _, rw, sink, _ := fixture(t)
			calls := 0
			rw.rewrite = func(request screenwire.Record, reply []byte) []byte {
				calls++
				if request.Kind != screenwire.Hello {
					t.Fatal("probe acquired or sent", request.Kind)
				}
				return reply
			}
			alterProbeExpectation(c, change)
			err := c.Probe()
			if change == "none" && err != nil {
				t.Fatal(err)
			}
			if change != "none" && (!errors.Is(err, ErrResync) || c.bound) {
				t.Fatal(err, c.bound)
			}
			if calls != 1 || sink.commits != 0 {
				t.Fatal(calls, sink.commits)
			}
		})
	}
}

func TestProbePropagatesFramingAndRemoteFailures(t *testing.T) {
	c, _, rw, _, _ := fixture(t)
	rw.dropKind = screenwire.Hello
	if err := c.Probe(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal(err)
	}
	c, _, rw, _, _ = fixture(t)
	rw.rewrite = func(_ screenwire.Record, reply []byte) []byte {
		r, err := screenwire.Decode(reply)
		if err != nil {
			t.Fatal(err)
		}
		r.Payload[1] = byte(screenwire.CodeConfig)
		out := make([]byte, screenwire.MaxRecord)
		n, err := screenwire.Encode(out, r)
		if err != nil {
			t.Fatal(err)
		}
		return out[:n]
	}
	var remote RemoteError
	if err := c.Probe(); !errors.As(err, &remote) || remote.Status.Code != screenwire.CodeConfig {
		t.Fatal(err)
	}
}

func alterProbeExpectation(c *Client, change string) {
	switch change {
	case "boot":
		c.lease.Boot[0]++
	case "generation":
		c.lease.Generation++
	case "caps":
		c.caps.Profile++
	}
}

func TestProbeRequiresReconciliationAndBoundConnection(t *testing.T) {
	c, _, rw, _, frame := fixture(t)
	rw.loseCommit = true
	if err := c.Send(frame); err == nil {
		t.Fatal("expected lost ACK")
	}
	if err := c.Probe(); !errors.Is(err, ErrPending) {
		t.Fatal(err)
	}
	c.pending.ID = 0
	if err := c.Probe(); !errors.Is(err, ErrConnection) {
		t.Fatal(err)
	}
}
