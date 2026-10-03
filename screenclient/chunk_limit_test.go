package screenclient

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestChunkLimitValidation(t *testing.T) {
	for _, limit := range []uint16{0, screenwire.MaxPayload + 1} {
		if _, err := NewWithMaxChunk(bytes.NewReader(nil), limit); !errors.Is(err, screenwire.ErrRecord) {
			t.Fatal(limit, err)
		}
	}
	if _, err := NewWithMaxChunk(nil, 480); !errors.Is(err, screenwire.ErrRecord) {
		t.Fatal(err)
	}
}

func TestChunkLimitPreservesNegotiationAndReconnect(t *testing.T) {
	for _, limit := range []uint16{1, 7, 10, screenwire.MaxPayload} {
		_, d, original, sink, frame := fixture(t)
		if err := original.connection.Disconnect(); err != nil {
			t.Fatal(err)
		}
		c, err := NewWithMaxChunk(bytes.NewReader(bytes.Repeat([]byte{3}, 64)), limit)
		if err != nil {
			t.Fatal(err)
		}
		for attempt := 0; attempt < 2; attempt++ {
			rw := &transport{connection: d.Open(), now: time.Duration(attempt+2) * time.Second}
			assertLimitedTransfer(t, c, rw, sink, frame, limit)
		}
		if sink.commits != 2 {
			t.Fatal(sink.commits)
		}
	}
}

func assertLimitedTransfer(t *testing.T, c *Client, rw *transport, sink *recordingSink, frame display.Frame, limit uint16) {
	t.Helper()
	largest := 0
	rw.rewrite = func(request screenwire.Record, reply []byte) []byte {
		if request.Kind == screenwire.Data {
			largest = max(largest, len(request.Payload))
		}
		return reply
	}
	caps, err := c.Connect(rw)
	if err != nil || caps.MaxChunk != 10 {
		t.Fatal("negotiated device capabilities changed", caps, err)
	}
	if err = c.Send(frame); err != nil || largest != min(int(limit), 10) {
		t.Fatal(limit, largest, err)
	}
	for _, plane := range sink.planes {
		if !bytes.Equal(plane, frame.Bytes()) {
			t.Fatal("chunk limit altered pixels")
		}
	}
	if err = rw.connection.Disconnect(); err != nil {
		t.Fatal(err)
	}
}
