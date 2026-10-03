package screenwire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"testing"
)

// Assemble the oracle independently: no Encode/Size/Decode in expected bytes.
func literal(r Record) []byte {
	b := make([]byte, 32+len(r.Payload))
	copy(b, "EPS2")
	b[4] = byte(r.Kind)
	b[5] = r.Pass
	binary.LittleEndian.PutUint16(b[6:8], uint16(len(r.Payload)))
	binary.LittleEndian.PutUint64(b[8:16], r.Epoch)
	binary.LittleEndian.PutUint64(b[16:24], r.ID)
	binary.LittleEndian.PutUint32(b[24:28], r.Offset)
	copy(b[32:], r.Payload)
	crc := crc32.NewIEEE()
	_, _ = crc.Write(b[:28])
	_, _ = crc.Write(b[32:])
	binary.LittleEndian.PutUint32(b[28:32], crc.Sum32())
	return b
}

func TestMaximumDataRecordExactBytes(t *testing.T) {
	r := Record{Kind: Data, Pass: 1, Epoch: 0x0807060504030201, ID: 9, Offset: 1234, Payload: bytes.Repeat([]byte{0xa5}, 1024)}
	b := make([]byte, 1056)
	n, err := Encode(b, r)
	if err != nil || n != 1056 || !bytes.Equal(b, literal(r)) {
		t.Fatal(n, err)
	}
	got, err := Decode(b)
	if err != nil || got.Kind != r.Kind || got.Offset != r.Offset || !bytes.Equal(got.Payload, r.Payload) {
		t.Fatal(got, err)
	}
	b[32] ^= 1
	if got.Payload[0] != b[32] {
		t.Fatal("Decode must borrow storage")
	}
	if _, err = Decode(b); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
}

func TestInvalidShapesRejectEvenWithCorrectCRC(t *testing.T) {
	for _, r := range []Record{
		{Kind: Hello, Epoch: 1}, {Kind: Hello, Payload: []byte{1}},
		{Kind: Acquire, Payload: make([]byte, 31)}, {Kind: Acquire, ID: 1, Payload: make([]byte, 32)},
		{Kind: Bind, Payload: make([]byte, 32)},
		{Kind: Begin, Epoch: 1, ID: 1, Payload: make([]byte, 31)},
		{Kind: Data, Epoch: 1, ID: 1, Pass: 2, Payload: []byte{1}},
		{Kind: Data, Epoch: 1, ID: 1, Payload: make([]byte, 1025)},
		{Kind: Data, Epoch: 1, ID: 1}, {Kind: Data, Epoch: 1, Payload: []byte{1}},
		{Kind: Commit, Epoch: 1, ID: 1}, {Kind: Query, Epoch: 1, ID: 1, Offset: 1, Payload: make([]byte, 32)},
		{Kind: Abort, Epoch: 1, ID: 1}, {Kind: Reply, Payload: make([]byte, 47)},
		{Kind: 0}, {Kind: Kind(255)},
	} {
		if _, err := Encode(make([]byte, 2048), r); !errors.Is(err, ErrRecord) {
			t.Fatal("encode", r, err)
		}
		if _, err := Decode(literal(r)); !errors.Is(err, ErrRecord) {
			t.Fatal("decode", r, err)
		}
	}
}

func TestTruncatedAndLegacyFramesReject(t *testing.T) {
	b := literal(Record{Kind: Hello})
	for n := 0; n < len(b); n++ {
		if _, err := Decode(b[:n]); !errors.Is(err, ErrRecord) {
			t.Fatal(n, err)
		}
	}
	if _, err := Encode(b[:31], Record{Kind: Hello}); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
	if _, err := Decode(append(b, 0)); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
	copy(b, "EPS1")
	if _, err := Size(b); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
}

func TestControlShapesRoundtrip(t *testing.T) {
	for _, r := range []Record{
		{Kind: Hello}, {Kind: Acquire, Payload: make([]byte, 32)},
		{Kind: Bind, Epoch: 1, Payload: make([]byte, 32)},
		{Kind: Begin, Epoch: 1, ID: 2, Payload: make([]byte, 32)},
		{Kind: Commit, Epoch: 1, ID: 2, Payload: make([]byte, 32)},
		{Kind: Query, Epoch: 1, ID: 2, Payload: make([]byte, 32)},
		{Kind: Abort, Epoch: 1},
	} {
		b := make([]byte, MaxRecord)
		n, err := Encode(b, r)
		if err != nil {
			t.Fatal(r, err)
		}
		if _, err = Decode(b[:n]); err != nil {
			t.Fatal(r, err)
		}
	}
}
