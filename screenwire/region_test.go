package screenwire

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
)

func testRegion() RegionBegin {
	return RegionBegin{Baseline: [32]byte{1}, Old: sha256.Sum256([]byte{0, 1, 2, 3}),
		New: sha256.Sum256([]byte{4, 5, 6, 7}), Left: 240, Top: 254, Right: 256, Bottom: 256,
		Priority: refreshpolicy.Urgent, Policy: refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}}
}

func TestRegionBeginRoundTripAndMetadataBinding(t *testing.T) {
	want := testRegion()
	b, err := EncodeRegionBegin(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeRegionBegin(b[:], 800, 480)
	if err != nil || got != want {
		t.Fatal(got, err)
	}
	for i := range b {
		bad := b
		bad[i] ^= 1
		if _, err := DecodeRegionBegin(bad[:], 800, 480); err == nil {
			t.Fatalf("unbound byte %d", i)
		}
	}
	for n := 0; n < len(b); n++ {
		if _, err := DecodeRegionBegin(b[:n], 800, 480); err == nil {
			t.Fatalf("truncation %d", n)
		}
	}
	if _, err := DecodeRegionBegin(append(b[:], 0), 800, 480); err == nil {
		t.Fatal("extra byte accepted")
	}
}

func TestRegionRecordShape(t *testing.T) {
	b, err := EncodeRegionBegin(testRegion())
	if err != nil {
		t.Fatal(err)
	}
	r := Record{Kind: BeginRegion, Epoch: 1, ID: 2, Payload: b[:]}
	var wire [MaxRecord]byte
	n, err := Encode(wire[:], r)
	if err != nil || n != HeaderSize+RegionBeginSize {
		t.Fatal(n, err)
	}
	if got, err := Decode(wire[:n]); err != nil || got.Kind != BeginRegion {
		t.Fatal(got, err)
	}
	for _, bad := range []Record{
		{Kind: BeginRegion, ID: 2, Payload: b[:]},
		{Kind: BeginRegion, Epoch: 1, Payload: b[:]},
		{Kind: BeginRegion, Epoch: 1, ID: 2, Payload: b[:147]},
		{Kind: BeginRegion, Epoch: 1, ID: 2, Pass: 1, Payload: b[:]},
		{Kind: BeginRegion, Epoch: 1, ID: 2, Offset: 1, Payload: b[:]},
	} {
		if _, err := Encode(wire[:], bad); err == nil {
			t.Fatal("bad shape", bad)
		}
	}
}

func TestRegionBeginRejectsInvalidFields(t *testing.T) {
	for _, change := range []func(*RegionBegin){
		func(r *RegionBegin) { r.Baseline = [32]byte{} },
		func(r *RegionBegin) { r.Left = r.Right },
		func(r *RegionBegin) { r.Top = r.Bottom },
		func(r *RegionBegin) { r.Left++ },
		func(r *RegionBegin) { r.Right++ },
		func(r *RegionBegin) { r.Priority = 2 },
		func(r *RegionBegin) { r.Policy.Normal = 0 },
		func(r *RegionBegin) { r.Policy.Urgent = time.Nanosecond },
	} {
		r := testRegion()
		change(&r)
		if _, err := EncodeRegionBegin(r); err == nil {
			t.Fatal(r)
		}
	}
	b, err := EncodeRegionBegin(testRegion())
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range [][2]uint16{{255, 480}, {800, 255}, {0, 0}} {
		if _, err := DecodeRegionBegin(b[:], size[0], size[1]); err == nil {
			t.Fatal("outside display", size)
		}
	}
}
