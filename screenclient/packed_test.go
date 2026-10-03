package screenclient

import (
	"bytes"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func packedFixture(t *testing.T, flags uint16, pixels []byte) (*Client, *transport, *recordingSink, display.Frame) {
	t.Helper()
	sink := &recordingSink{}
	caps := screenwire.Capabilities{Width: 800, Height: 480, Stride: 100, MaxChunk: 1000, Passes: 2,
		Format: screenwire.Mono1, Features: flags, Profile: 1, ProfileVersion: 1, MinimumFullMS: 1000}
	d, err := screenlink.New(caps, [16]byte{1}, sink, time.Second, 2*time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	rw := &transport{connection: d.Open(), now: time.Second}
	c, err := New(bytes.NewReader(bytes.Repeat([]byte{1}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Connect(rw); err != nil {
		t.Fatal(err)
	}
	f, err := display.NewFrame(display.Size{Width: 800, Height: 480}, 100, pixels)
	if err != nil {
		t.Fatal(err)
	}
	return c, rw, sink, f
}

func TestCompressionNegotiationReplaysExactPlanesAndReducesWholeWireBytes(t *testing.T) {
	for _, fixture := range []string{"solid", "noise", "mixed"} {
		pixels := packedPixels(fixture)
		var totals [2]int
		for mode, flags := range []uint16{screenwire.RawFull, screenwire.RawFull | screenwire.FeaturePackBits} {
			c, rw, sink, frame := packedFixture(t, flags, pixels)
			if err := c.Send(frame); err != nil {
				t.Fatal(err)
			}
			for _, plane := range sink.planes {
				if !bytes.Equal(plane, pixels) {
					t.Fatal("decoded plane differs")
				}
			}
			if sink.commits != 1 {
				t.Fatal(sink.commits)
			}
			totals[mode] = rw.wireBytes
		}
		if totals[1] > totals[0] || (fixture != "noise" && totals[1] == totals[0]) {
			t.Fatal(fixture, totals)
		}
		// Counts include Hello/Acquire/Bind/Begin/Commit and EVERY data ACK.
		t.Logf("%s EPS2 complete USB exchange: raw=%d packed=%d bytes", fixture, totals[0], totals[1])
	}
}

func packedPixels(kind string) []byte {
	p := make([]byte, 48000)
	for i := range p {
		if kind == "noise" || (kind == "mixed" && (i/1000)%2 == 0) {
			p[i] = byte((i*37 + i/251) % 256)
		}
	}
	return p
}
