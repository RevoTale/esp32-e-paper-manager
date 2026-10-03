package screendelivery

import (
	"bytes"
	"testing"
)

func FuzzRegionPlanReplaysSnapshot(f *testing.F) {
	f.Add([]byte{0, 255, 170, 85}, uint8(4), uint8(8))
	f.Add([]byte{}, uint8(1), uint8(1))
	f.Fuzz(func(t *testing.T, data []byte, w, h uint8) {
		width, height := (int(w)%8+2)*8, int(h)%16+2
		base, target := regionFrame(t, width, height, width/8), regionFrame(t, width, height, width/8+1)
		if len(data) > 0 {
			for i := range base.Bytes() {
				base.Bytes()[i] = data[i%len(data)]
			}
			for i := range target.Bytes() {
				target.Bytes()[i] = data[(i+1)%len(data)]
			}
		}
		before, after := bytes.Clone(base.Bytes()), bytes.Clone(target.Bytes())
		p, err := PlanRegion(base, target, RegionRules{16, 2, width / 8 * height})
		if err != nil {
			t.Fatal(err)
		}
		assertRegionReplay(t, base, target, p)
		if !bytes.Equal(before, base.Bytes()) || !bytes.Equal(after, target.Bytes()) {
			t.Fatal("mutated inputs")
		}
	})
}
