package streamwire

import "testing"

func TestRoundTripAndCorruption(t *testing.T) {
	var wire [MaxRecord]byte
	r := Record{Kind: Data, Epoch: 1, ID: 2, Pass: 1, Offset: 100, Payload: []byte{128, 1}}
	n, err := Encode(wire[:], r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(wire[:n])
	if err != nil || got.ID != 2 || got.Offset != 100 {
		t.Fatal(got, err)
	}
	for i := 0; i < n; i++ {
		wire[i] ^= 1
		if _, err = Decode(wire[:n]); err == nil {
			t.Fatal("corruption accepted", i)
		}
		wire[i] ^= 1
	}
	for i := 0; i < n; i++ {
		if _, err = Decode(wire[:i]); err == nil {
			t.Fatal("truncation", i)
		}
	}
}
