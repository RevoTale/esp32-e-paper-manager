package screenwire

import (
	"bytes"
	"testing"
)

func TestPackedOperationValuesAreAdditive(t *testing.T) {
	if DataPacked != 11 || Reply != 9 || Health != 10 {
		t.Fatal(DataPacked, Reply, Health)
	}
}

func TestPackedDataKeepsDecodedCoordinates(t *testing.T) {
	r := Record{Kind: DataPacked, Epoch: 3, ID: 8, Pass: 1, Offset: 1000, Payload: []byte{128, 0, 129, 255}}
	data := make([]byte, MaxRecord)
	n, err := Encode(data, r)
	if err != nil || !bytes.Equal(data[:n], literal(r)) {
		t.Fatal(n, err)
	}
	got, err := Decode(data[:n])
	if err != nil || got.Kind != DataPacked || got.Offset != 1000 || !bytes.Equal(got.Payload, r.Payload) {
		t.Fatal(got, err)
	}
	for _, bad := range []Record{
		{Kind: DataPacked, Epoch: 1, ID: 1, Payload: []byte{1, 0, 0}},
		{Kind: DataPacked, Epoch: 1, ID: 1, Pass: 2, Payload: r.Payload},
		{Kind: DataPacked, Epoch: 1, Payload: r.Payload},
	} {
		if _, err := Encode(data, bad); err == nil {
			t.Fatal(bad)
		}
	}
}

func TestOnlyKnownFeaturesAreOptionalInCapabilities(t *testing.T) {
	c := Capabilities{Width: 8, Height: 1, Stride: 1, MaxChunk: 1000, Passes: 2, Format: Mono1, Profile: 1, ProfileVersion: 1, MinimumFullMS: 1}
	for _, flags := range []uint16{RawFull, RawFull | FeaturePackBits, RawFull | FeatureRefreshPolicy, RawFull | FeaturePackBits | FeatureRefreshPolicy} {
		c.Features = flags
		var data [CapabilitiesSize]byte
		if err := EncodeCapabilities(data[:], c); err != nil {
			t.Fatal(err)
		}
		if got, err := DecodeCapabilities(data[:]); err != nil || got != c {
			t.Fatal(got, err)
		}
	}
	for _, flags := range []uint16{0, 1, 2, FeaturePackBits, FeatureRefreshPolicy, RawFull | 16, 0xffff} {
		c.Features = flags
		if err := EncodeCapabilities(make([]byte, CapabilitiesSize), c); err == nil {
			t.Fatal(flags)
		}
	}
}
