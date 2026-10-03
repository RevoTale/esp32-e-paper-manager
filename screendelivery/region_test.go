package screendelivery

import (
	"bytes"
	"errors"
	"image"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func regionFrame(t *testing.T, width, height, stride int) display.Frame {
	t.Helper()
	f, err := display.NewFrame(display.Size{Width: width, Height: height}, stride, make([]byte, stride*height))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestRegionPlanExpandsAtEveryEdgeAndReplays(t *testing.T) {
	for _, point := range []image.Point{{0, 0}, {31, 0}, {0, 7}, {31, 7}, {15, 3}} {
		base, target := regionFrame(t, 32, 8, 4), regionFrame(t, 32, 8, 5)
		if err := target.SetPixel(point.X, point.Y, display.Black); err != nil {
			t.Fatal(err)
		}
		p, err := PlanRegion(base, target, RegionRules{MinWidth: 16, MinHeight: 2, MaxBytes: 32})
		if err != nil {
			t.Fatal(err)
		}
		if p.Bounds.Dx() != 16 || p.Bounds.Dy() != 2 || !point.In(p.Bounds) {
			t.Fatal(p.Bounds)
		}
		assertRegionReplay(t, base, target, p)
	}
}

func assertRegionReplay(t *testing.T, base, target display.Frame, p RegionPlan) {
	t.Helper()
	out := regionFrame(t, base.Size().Width, base.Size().Height, base.Stride())
	copy(out.Bytes(), base.Bytes())
	for y := p.Bounds.Min.Y; y < p.Bounds.Max.Y; y++ {
		for x := p.Bounds.Min.X; x < p.Bounds.Max.X; x++ {
			i := (y-p.Bounds.Min.Y)*(p.Bounds.Dx()/8) + (x-p.Bounds.Min.X)/8
			mask := byte(0x80 >> uint((x-p.Bounds.Min.X)%8))
			if (p.Old[i]&mask != 0) != (base.Pixel(x, y) == display.Black) {
				t.Fatal("wrong old pixels")
			}
			color := display.White
			if p.New[i]&mask != 0 {
				color = display.Black
			}
			if err := out.SetPixel(x, y, color); err != nil {
				t.Fatal(err)
			}
		}
	}
	for y := 0; y < target.Size().Height; y++ {
		for x := 0; x < target.Size().Width; x++ {
			if out.Pixel(x, y) != target.Pixel(x, y) {
				t.Fatalf("lost damage %d,%d", x, y)
			}
		}
	}
}

func TestRegionPlanIncludesEraseAndDisconnectedDamage(t *testing.T) {
	base, target := regionFrame(t, 32, 8, 4), regionFrame(t, 32, 8, 4)
	base.Bytes()[0] = 255
	target.Bytes()[31] = 255
	before, after := bytes.Clone(base.Bytes()), bytes.Clone(target.Bytes())
	p, err := PlanRegion(base, target, RegionRules{16, 2, 32})
	if err != nil || p.Bounds != image.Rect(0, 0, 32, 8) {
		t.Fatal(p, err)
	}
	assertRegionReplay(t, base, target, p)
	p.Old[0] ^= 255
	p.New[0] ^= 255
	if !bytes.Equal(before, base.Bytes()) || !bytes.Equal(after, target.Bytes()) {
		t.Fatal("mutated borrowed input")
	}
	if _, err := PlanRegion(base, target, RegionRules{16, 2, 31}); !errors.Is(err, ErrRegionBudget) {
		t.Fatal(err)
	}
}

func TestRegionPlanNoChangeAndInvalidInputs(t *testing.T) {
	f := regionFrame(t, 32, 8, 4)
	p, err := PlanRegion(f, f, RegionRules{16, 2, 32})
	if err != nil || !p.Bounds.Empty() || p.Old != nil || p.New != nil {
		t.Fatal(p, err)
	}
	for _, rules := range []RegionRules{{0, 2, 32}, {7, 2, 32}, {40, 2, 32}, {16, 0, 32}, {16, 9, 32}, {16, 2, 0}, {16, 2, 3}} {
		if _, err := PlanRegion(f, f, rules); !errors.Is(err, ErrRegionInput) {
			t.Fatal(rules, err)
		}
	}
	for _, other := range []display.Frame{{}, regionFrame(t, 24, 8, 3), regionFrame(t, 31, 8, 4)} {
		if _, err := PlanRegion(f, other, RegionRules{16, 2, 32}); !errors.Is(err, ErrRegionInput) {
			t.Fatal(err)
		}
	}
}
