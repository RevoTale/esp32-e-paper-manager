package screenwire

import (
	"encoding/binary"
	"errors"
	"testing"
)

func TestPanelStatusWireRoundTrip(t *testing.T) {
	want := PanelStatus{Version: 1, State: 2, Phase: 6, Step: 17, Cycle: 7, ElapsedMS: 1234,
		Waits: [3]BusySamples{{3, 2}, {500, 499}, {1, 0}}}
	var data [PanelStatusSize]byte
	if err := EncodePanelStatus(data[:], want); err != nil {
		t.Fatal(err)
	}
	if len(data) != 36 || data[0] != 1 || data[1] != 2 || binary.LittleEndian.Uint32(data[16:20]) != 2 {
		t.Fatal(data)
	}
	if got, err := DecodePanelStatus(data[:]); err != nil || got != want {
		t.Fatal(got, err)
	}
	assertPanelReply(t, data[:], want)
}

func assertPanelReply(t *testing.T, data []byte, want PanelStatus) {
	t.Helper()
	request := Record{Kind: PanelTrace}
	body := make([]byte, StatusSize+PanelStatusSize)
	if err := EncodeStatus(body[:StatusSize], Status{Operation: PanelTrace, Boot: [16]byte{1}}); err != nil {
		t.Fatal(err)
	}
	copy(body[StatusSize:], data)
	var record [MaxRecord]byte
	n, err := Encode(record[:], Record{Kind: Reply, Payload: body})
	if err != nil {
		t.Fatal(err)
	}
	r, err := Decode(record[:n])
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseReply(r, request)
	if err != nil || got.Panel != want {
		t.Fatal(got, err)
	}
	if _, err = ParseReply(r, Record{Kind: Health}); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
}

func TestPanelStatusRejectsMalformed(t *testing.T) {
	for _, value := range []PanelStatus{{}, {Version: 2}, {Version: 1, State: 4},
		{Version: 1, State: 1, Cycle: 1, Waits: [3]BusySamples{{1, 2}}},
		{Version: 1, Cycle: 1}, {Version: 1, State: 1}} {
		if err := EncodePanelStatus(make([]byte, PanelStatusSize), value); !errors.Is(err, ErrRecord) {
			t.Fatal(value, err)
		}
	}
	for _, n := range []int{0, PanelStatusSize - 1, PanelStatusSize + 1} {
		if err := EncodePanelStatus(make([]byte, n), PanelStatus{Version: 1}); !errors.Is(err, ErrRecord) {
			t.Fatal(n, err)
		}
		if _, err := DecodePanelStatus(make([]byte, n)); !errors.Is(err, ErrRecord) {
			t.Fatal(n, err)
		}
	}
	for _, request := range []Record{{Kind: PanelTrace, Epoch: 1}, {Kind: PanelTrace, ID: 1},
		{Kind: PanelTrace, Pass: 1}, {Kind: PanelTrace, Offset: 1}, {Kind: PanelTrace, Payload: []byte{1}}} {
		if _, err := Decode(literal(request)); !errors.Is(err, ErrRecord) {
			t.Fatal(request, err)
		}
	}
	data := make([]byte, PanelStatusSize)
	data[0] = 2
	if _, err := DecodePanelStatus(data); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
}
