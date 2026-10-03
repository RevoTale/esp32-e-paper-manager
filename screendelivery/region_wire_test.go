package screendelivery

import (
	"crypto/sha256"
	"image"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestRegionWireBindsSnapshots(t *testing.T) {
	p := RegionPlan{Bounds: image.Rect(240, 254, 256, 256), Old: []byte{1, 2, 3, 4}, New: []byte{5, 6, 7, 8}}
	baseline := sha256.Sum256([]byte("confirmed"))
	o := refreshpolicy.Options{Mode: refreshpolicy.Partial, Priority: refreshpolicy.Urgent}
	policy := refreshpolicy.Policy{Normal: 2 * time.Second, Urgent: time.Second}
	r, err := p.Wire(baseline, o, policy)
	if err != nil {
		t.Fatal(err)
	}
	want := screenwire.RegionBegin{Baseline: baseline, Old: sha256.Sum256(p.Old), New: sha256.Sum256(p.New),
		Left: 240, Top: 254, Right: 256, Bottom: 256, Priority: o.Priority, Policy: policy}
	if r != want {
		t.Fatal(r, err)
	}
	b, err := screenwire.EncodeRegionBegin(r)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := screenwire.DecodeRegionBegin(b[:], 800, 480)
	if err != nil || decoded != r {
		t.Fatal(decoded, err)
	}
}

func TestRegionWireRejectsNarrowing(t *testing.T) {
	p := RegionPlan{Bounds: image.Rect(240, 254, 256, 256), Old: []byte{1, 2, 3, 4}, New: []byte{5, 6, 7, 8}}
	baseline := sha256.Sum256([]byte("confirmed"))
	o := refreshpolicy.Options{Mode: refreshpolicy.Partial}
	policy := refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}
	for _, bounds := range []image.Rectangle{image.Rect(-8, 0, 8, 2), image.Rect(0, -1, 16, 1), image.Rect(65536, 0, 65552, 2), image.Rect(0, 65536, 16, 65538), image.Rect(1, 0, 17, 2), {}} {
		bad := p
		bad.Bounds = bounds
		if _, err := bad.Wire(baseline, o, policy); err == nil {
			t.Fatal("invalid geometry", bounds)
		}
	}
	p.New = nil
	if _, err := p.Wire(baseline, o, policy); err == nil {
		t.Fatal("invalid crop length")
	}
}

func TestRegionWireRejectsUnknownIdentityAndWrongMode(t *testing.T) {
	p := RegionPlan{Bounds: image.Rect(0, 0, 8, 1), Old: []byte{0}, New: []byte{1}}
	policy := refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}
	for _, o := range []refreshpolicy.Options{{}, {Mode: refreshpolicy.Full}, {Mode: refreshpolicy.Partial, Priority: 2}} {
		if _, err := p.Wire([32]byte{1}, o, policy); err == nil {
			t.Fatal(o)
		}
	}
	if _, err := p.Wire([32]byte{}, refreshpolicy.Options{Mode: refreshpolicy.Partial}, policy); err == nil {
		t.Fatal("unknown baseline")
	}
}
