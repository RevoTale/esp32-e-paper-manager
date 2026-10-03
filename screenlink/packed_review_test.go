package screenlink

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestPackedReviewForeignIdentityCannotDecodeOrAbortReadyStaging(t *testing.T) {
	for _, mismatch := range []string{"generation", "id", "unbound"} {
		t.Run(mismatch, func(t *testing.T) {
			d, c, sink := fixture(t)
			d.caps.Features |= screenwire.FeaturePackBits
			bind(t, c, 1, true)
			upload(t, c, 1)
			for i := range d.decoded {
				d.decoded[i] = 0x5a
			}
			request := screenwire.Record{Kind: screenwire.DataPacked, Epoch: 1, ID: 1, Payload: []byte{1, 0, 0, 128}}
			target, code := c, screenwire.CodeLease
			switch mismatch {
			case "generation":
				request.Epoch++
			case "id":
				request.ID++
				code = screenwire.CodeState
			case "unbound":
				target = d.Open()
			}
			if got := exchange(t, target, request, time.Second); got.Code != code {
				t.Fatal(got)
			}
			if !bytes.Equal(d.decoded[:], bytes.Repeat([]byte{0x5a}, screenwire.MaxPayload)) {
				t.Fatal("foreign identity decoded")
			}
			wantCalls(t, sink, [4]int{1, 2, 0, 0})
			digest := sha256.Sum256([]byte{128})
			if got := exchange(t, c, screenwire.Record{Kind: screenwire.Commit, Epoch: 1, ID: 1, Payload: digest[:]}, time.Second); got.Code != screenwire.CodeOK {
				t.Fatal(got)
			}
			wantCalls(t, sink, [4]int{1, 2, 1, 0})
		})
	}
}

type packedReviewSink struct {
	sink
	abortErr error
}

func (s *packedReviewSink) Abort() error { s.aborts++; return s.abortErr }

func packedReviewFixture(t *testing.T) (*Device, *Connection, *packedReviewSink) {
	t.Helper()
	sink := &packedReviewSink{}
	caps := screenwire.Capabilities{Width: 9, Height: 1, Stride: 2, MaxChunk: 2, Passes: 2, Format: screenwire.Mono1,
		Features: screenwire.RawFull | screenwire.FeaturePackBits, Profile: 1, ProfileVersion: 1, MinimumFullMS: 1000}
	d, err := New(caps, [16]byte{1}, sink, time.Second, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := d.Open()
	bind(t, c, 1, true)
	digest := sha256.Sum256([]byte{0, 0})
	if got := exchange(t, c, screenwire.Record{Kind: screenwire.Begin, Epoch: 1, ID: 1, Payload: digest[:]}, time.Second); got.Code != screenwire.CodeOK {
		t.Fatal(got)
	}
	return d, c, sink
}

func TestPackedReviewDecodedPaddingRejectsBeforeSinkWrite(t *testing.T) {
	_, c, sink := packedReviewFixture(t)
	r := screenwire.Record{Kind: screenwire.DataPacked, Epoch: 1, ID: 1, Payload: []byte{2, 0, 1, 0, 1}}
	if got := exchange(t, c, r, time.Second); got.Code != screenwire.CodeChunk {
		t.Fatal(got)
	}
	wantCalls(t, &sink.sink, [4]int{1, 0, 0, 1})
}

func TestPackedReviewPartialDecodeDoesNotPublishAndPreservesAbortFailure(t *testing.T) {
	d, c, sink := packedReviewFixture(t)
	sink.abortErr = errors.New("synthetic cleanup failure")
	d.decoded[0], d.decoded[1] = 0x5a, 0x5a
	r := screenwire.Record{Kind: screenwire.DataPacked, Epoch: 1, ID: 1, Payload: []byte{2, 0, 0, 0, 128}}
	_, err := c.data(r, time.Second)
	if !errors.Is(err, streamrx.ErrChunk) || !errors.Is(err, sink.abortErr) {
		t.Fatal(err)
	}
	if d.decoded[0] != 0 || d.decoded[1] != 0x5a {
		t.Fatal("fixture did not exercise partial decode")
	}
	wantCalls(t, &sink.sink, [4]int{1, 0, 0, 1})
}

func TestPackedReviewDecodeBoundaryUsesOnlyFixedDeviceStorage(t *testing.T) {
	d, c, _ := packedReviewFixture(t)
	payload := []byte{2, 0, 255, 0}
	if got := testing.AllocsPerRun(100, func() {
		pixels, err := c.unpack(payload)
		if err != nil || len(pixels) != 2 || &pixels[0] != &d.decoded[0] {
			t.Fatal(err)
		}
	}); got != 0 {
		t.Fatal("per-chunk allocation", got)
	}
}
