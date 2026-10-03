package interop

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestGoRegionSenderToNativePanel(t *testing.T) {
	testNativeRegion(t, false)
}

func TestGoRegionLostCommitQueriesNativePanel(t *testing.T) {
	testNativeRegion(t, true)
}

func testNativeRegion(t *testing.T, lost bool) {
	t.Helper()
	peer := startNativePeerNamed(t, "EP_RECEIVER_CLI", "--region")
	stream := &lostCommitStream{ReadWriter: peerStream{peer.output, peer.input}}
	client, err := screenclient.New(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caps, err := client.Connect(stream)
	if err != nil {
		t.Fatal(err)
	}
	if caps.Features&screenwire.FeatureRegion == 0 {
		t.Fatal("missing region capability")
	}
	if err := client.Send(testFrame(t, 0xa5)); err != nil {
		t.Fatal(err)
	}
	old, next := bytes.Repeat([]byte{0xa5}, 4), bytes.Repeat([]byte{0x5a}, 4)
	r := screenwire.RegionBegin{Baseline: client.Baseline(), Old: sha256.Sum256(old), New: sha256.Sum256(next),
		Left: 240, Top: 254, Right: 256, Bottom: 256,
		Policy: refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}}
	stream.enabled = lost
	err = client.SendRegion(r, old, next)
	if lost {
		reconcileNativeRegion(t, client, stream, err)
	} else if err != nil {
		t.Fatal(err)
	}
	b, err := screenwire.EncodeRegionBegin(r)
	if err != nil {
		t.Fatal(err)
	}
	if client.Pending() || client.Baseline() != [32]byte(b[:32]) {
		t.Fatal("missing confirmed region identity")
	}
	// Native peer asserts exact SPI polarity, window, plane bytes and refresh count.
	peer.finish()
}
