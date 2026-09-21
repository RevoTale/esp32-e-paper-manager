package screenlink

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestPackedDataIsDecodedBeforeWriteAndUsesNormalProgress(t *testing.T) {
	d, c, sink := fixture(t)
	d.caps.Features |= screenwire.FeaturePackBits
	bind(t, c, 1, true)
	digest := sha256.Sum256([]byte{128})
	r := screenwire.Record{Kind: screenwire.Begin, Epoch: 1, ID: 1, Payload: digest[:]}
	if got := exchange(t, c, r, time.Second); got.Code != screenwire.CodeOK {
		t.Fatal(got)
	}
	for pass := uint8(0); pass < 2; pass++ {
		r = screenwire.Record{Kind: screenwire.DataPacked, Epoch: 1, ID: 1, Pass: pass, Payload: []byte{1, 0, 0, 128}}
		if got := exchange(t, c, r, time.Second); got.Code != screenwire.CodeOK || got.Pass != pass+1 {
			t.Fatal(got)
		}
	}
	r = screenwire.Record{Kind: screenwire.Commit, Epoch: 1, ID: 1, Payload: digest[:]}
	if got := exchange(t, c, r, time.Second); got.Code != screenwire.CodeOK || !got.CurrentImage {
		t.Fatal(got)
	}
	wantCalls(t, sink, [4]int{1, 2, 1, 0})
}

func TestBadPackedBlockRetiresOnlyItsBoundTransaction(t *testing.T) {
	for _, payload := range [][]byte{
		{0, 0, 0, 128}, {2, 0, 255, 128}, {1, 0, 128, 128},
		{1, 0, 1, 128}, {1, 0, 255, 128}, {1, 0, 0, 128, 0, 128},
	} {
		d, c, sink := fixture(t)
		d.caps.Features |= screenwire.FeaturePackBits
		bind(t, c, 1, true)
		upload(t, c, 1) // Ready: malformed extra input must forbid later Commit.
		r := screenwire.Record{Kind: screenwire.DataPacked, Epoch: 1, ID: 1, Payload: payload}
		if got := exchange(t, c, r, time.Second); got.Code != screenwire.CodeChunk {
			t.Fatal(got)
		}
		digest := sha256.Sum256([]byte{128})
		r = screenwire.Record{Kind: screenwire.Commit, Epoch: 1, ID: 1, Payload: digest[:]}
		if got := exchange(t, c, r, time.Second); got.Code == screenwire.CodeOK {
			t.Fatal("commit after rejected block", got)
		}
		wantCalls(t, sink, [4]int{1, 2, 0, 1})
	}
}

func TestPackedDataRequiresCapabilityBindingAndID(t *testing.T) {
	_, c, sink := fixture(t) // Deliberately raw-only.
	r := screenwire.Record{Kind: screenwire.DataPacked, Epoch: 1, ID: 1, Payload: []byte{1, 0, 0, 128}}
	if got := exchange(t, c, r, time.Second); got.Code != screenwire.CodeLease {
		t.Fatal(got)
	}
	bind(t, c, 1, true)
	upload(t, c, 1)
	r.ID = 2
	if got := exchange(t, c, r, time.Second); got.Code != screenwire.CodeState {
		t.Fatal(got)
	}
	wantCalls(t, sink, [4]int{1, 2, 0, 0})
	r.ID = 1
	if got := exchange(t, c, r, time.Second); got.Code != screenwire.CodeChunk {
		t.Fatal(got)
	}
	wantCalls(t, sink, [4]int{1, 2, 0, 1})
}
