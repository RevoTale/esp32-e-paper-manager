package interop

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestCRecordCodecMatchesGo(t *testing.T) {
	records := []screenwire.Record{
		{Kind: screenwire.Hello}, {Kind: screenwire.Health}, {Kind: screenwire.PanelTrace},
		{Kind: screenwire.Acquire, Payload: make([]byte, 32)},
		{Kind: screenwire.Bind, Epoch: 0xffff, Payload: make([]byte, 32)},
		{Kind: screenwire.Begin, Epoch: 37, ID: 12, Payload: make([]byte, 32)},
		{Kind: screenwire.BeginRegion, Epoch: 37, ID: 12, Payload: make([]byte, screenwire.RegionBeginSize)},
		{Kind: screenwire.Commit, Epoch: 37, ID: 12, Payload: make([]byte, 32)},
		{Kind: screenwire.Query, Epoch: 37, ID: 12, Payload: make([]byte, 32)},
		{Kind: screenwire.Abort, Epoch: 37},
		{Kind: screenwire.Data, Epoch: 37, ID: 12, Pass: 1, Offset: 47000, Payload: bytes.Repeat([]byte{0x55}, 1000)},
		{Kind: screenwire.DataPacked, Epoch: 37, ID: 12, Payload: []byte{1, 0, 0, 255}},
	}
	for _, size := range []int{48, 88, 56, 84} {
		records = append(records, screenwire.Record{Kind: screenwire.Reply, Payload: make([]byte, size)})
	}
	var input, expected bytes.Buffer
	for _, record := range records {
		var wire [screenwire.MaxRecord]byte
		n, err := screenwire.Encode(wire[:], record)
		if err != nil {
			t.Fatal(err)
		}
		var length [2]byte
		binary.BigEndian.PutUint16(length[:], uint16(n))
		input.Write(length[:])
		input.Write(wire[:n])
		expected.Write(wire[:n])
	}
	path := os.Getenv("EP_WIRE_CLI")
	if path == "" {
		t.Fatal("EP_WIRE_CLI must name compiled C wire peer")
	}
	cmd := exec.Command(path)
	cmd.Stdin = &input
	actual, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected.Bytes()) {
		t.Fatal("C EPS2 output differs from Go")
	}
}
