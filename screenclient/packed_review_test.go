package screenclient

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func packedReviewClient(t *testing.T) (*Client, *screenlink.Device, *transport, *recordingSink, display.Frame) {
	t.Helper()
	sink := &recordingSink{}
	caps := screenwire.Capabilities{Width: 80, Height: 3, Stride: 10, MaxChunk: 10, Passes: 2, Format: screenwire.Mono1,
		Features: screenwire.RawFull | screenwire.FeaturePackBits, Profile: 1, ProfileVersion: 1, MinimumFullMS: 1000}
	d, err := screenlink.New(caps, [16]byte{1}, sink, time.Second, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	rw := &transport{connection: d.Open(), now: time.Second}
	c, err := New(bytes.NewReader(bytes.Repeat([]byte{1}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Connect(rw); err != nil {
		t.Fatal(err)
	}
	frame, err := display.NewFrame(display.Size{Width: 80, Height: 3}, 10, make([]byte, 30))
	if err != nil {
		t.Fatal(err)
	}
	return c, d, rw, sink, frame
}

func TestPackedReviewLostDataACKDoesNotReplayBeforeNewExplicitID(t *testing.T) {
	c, d, rw, sink, frame := packedReviewClient(t)
	rw.dropKind = screenwire.DataPacked
	if err := c.Send(frame); !errors.Is(err, io.ErrUnexpectedEOF) || !c.Pending() {
		t.Fatal(err)
	}
	assertOnlyOnePackedChunk(t, sink)
	rw = &transport{connection: d.Open(), now: time.Second}
	if _, err := c.Connect(rw); err != nil {
		t.Fatal(err)
	}
	assertPackedUnconfirmed(t, c)
	assertOnlyOnePackedChunk(t, sink)
	if err := c.Send(frame); err != nil {
		t.Fatal(err)
	}
	assertNewPackedUpload(t, c, sink, frame)
}

func assertOnlyOnePackedChunk(t *testing.T, sink *recordingSink) {
	t.Helper()
	if len(sink.planes[0]) != 10 || sink.commits != 0 {
		t.Fatal("expected only one staged packed chunk")
	}
}

func assertPackedUnconfirmed(t *testing.T, c *Client) {
	t.Helper()
	r, err := c.Reconcile()
	if err != nil || r.Confirmed || c.Pending() {
		t.Fatal(r, err)
	}
}

func assertNewPackedUpload(t *testing.T, c *Client, sink *recordingSink, frame display.Frame) {
	t.Helper()
	if c.next != 2 || sink.commits != 1 {
		t.Fatal(c.next, sink.commits)
	}
	for _, plane := range sink.planes {
		if !bytes.Equal(plane, frame.Bytes()) {
			t.Fatal("new complete upload differed")
		}
	}
}
