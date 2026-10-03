package interop

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestCRegionContentMatchesGo(t *testing.T) {
	r := screenwire.RegionBegin{Baseline: [32]byte{7}, Old: sha256.Sum256([]byte{0, 1, 2, 3}),
		New: sha256.Sum256([]byte{4, 5, 6, 7}), Left: 240, Top: 254, Right: 256, Bottom: 256,
		Priority: refreshpolicy.Urgent, Policy: refreshpolicy.Policy{Normal: time.Second, Urgent: 500 * time.Millisecond}}
	payload, err := screenwire.EncodeRegionBegin(r)
	if err != nil {
		t.Fatal(err)
	}
	var input, want bytes.Buffer
	appendRegionCase(t, &input, &want, payload, 800, 480)
	appendRegionCase(t, &input, &want, payload, 255, 480)
	appendRegionCase(t, &input, &want, payload, 800, 255)
	for i := range payload {
		bad := payload
		bad[i] ^= 1
		appendRegionCase(t, &input, &want, bad, 800, 480)
	}
	path := os.Getenv("EP_REGION_CLI")
	if path == "" {
		t.Fatal("EP_REGION_CLI must name compiled C region peer")
	}
	cmd := exec.Command(path)
	cmd.Stdin = &input
	got, err := cmd.Output()
	if err != nil || !bytes.Equal(got, want.Bytes()) {
		t.Fatalf("C/Go region mismatch: got %d bytes, want %d, error %v", len(got), want.Len(), err)
	}
}

func appendRegionCase(t *testing.T, input, want *bytes.Buffer, b [screenwire.RegionBeginSize]byte, width, height uint16) {
	t.Helper()
	var dimensions [4]byte
	binary.LittleEndian.PutUint16(dimensions[:2], width)
	binary.LittleEndian.PutUint16(dimensions[2:], height)
	input.Write(dimensions[:])
	input.Write(b[:])
	var expected [121]byte
	r, err := screenwire.DecodeRegionBegin(b[:], width, height)
	if err != nil {
		expected[0] = 1
	} else {
		copy(expected[1:117], b[32:])
		binary.LittleEndian.PutUint32(expected[117:], uint32(r.Right-r.Left)/8*uint32(r.Bottom-r.Top))
	}
	want.Write(expected[:])
}
