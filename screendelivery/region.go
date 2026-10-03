package screendelivery

import (
	"errors"
	"image"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

var (
	ErrRegionInput  = errors.New("screen region: incompatible frames or window rules")
	ErrRegionBudget = errors.New("screen region: damage exceeds partial budget")
)

// RegionRules comes from the selected adapter and operator policy, not HTML.
// Width is byte aligned. MaxBytes bounds each old/new packed plane allocation.
type RegionRules struct {
	MinWidth, MinHeight, MaxBytes int
}

func (r RegionRules) Validate(size display.Size) error {
	if !r.valid(size) {
		return ErrRegionInput
	}
	return nil
}

// RegionPlan owns opaque old/new crops of one coherent final scene pair.
// Empty bounds mean pixel-equivalent frames and no physical update is needed.
type RegionPlan struct {
	Bounds   image.Rectangle
	Old, New []byte
}

// PlanRegion compares immutable final pixels, never independent DOM patches.
// Adapted from the project's renderdiff byte-row scan for EPS2's single window.
// Panel minimum dimensions expand the damage using unchanged pixels from the
// same snapshots, preserving backgrounds and the confirmed timestamp corner.
// Non-byte-width displays explicitly require a different adapter or full path.
func PlanRegion(base, target display.Frame, rules RegionRules) (RegionPlan, error) {
	if !rules.valid(base.Size()) || base.Size() != target.Size() {
		return RegionPlan{}, ErrRegionInput
	}
	bounds := regionDamage(base, target)
	if bounds.Empty() {
		return RegionPlan{}, nil
	}
	bounds.Min.X, bounds.Max.X = expandSpan(bounds.Min.X, bounds.Max.X, rules.MinWidth, base.Size().Width)
	bounds.Min.Y, bounds.Max.Y = expandSpan(bounds.Min.Y, bounds.Max.Y, rules.MinHeight, base.Size().Height)
	if bounds.Dx()/8*bounds.Dy() > rules.MaxBytes {
		return RegionPlan{}, ErrRegionBudget
	}
	return RegionPlan{Bounds: bounds, Old: regionCrop(base, bounds), New: regionCrop(target, bounds)}, nil
}

func (r RegionRules) valid(size display.Size) bool {
	return validRegionViewport(size) &&
		r.MinWidth > 0 && r.MinWidth%8 == 0 && r.MinWidth <= size.Width && r.MinHeight > 0 && r.MinHeight <= size.Height &&
		r.MaxBytes >= r.MinWidth/8*r.MinHeight
}

func validRegionViewport(size display.Size) bool {
	return size.Width > 0 && size.Height > 0 && size.Width <= 65535 && size.Height <= 65535 && size.Width%8 == 0
}

func regionDamage(base, target display.Frame) image.Rectangle {
	var bounds image.Rectangle
	for y := 0; y < base.Size().Height; y++ {
		first, last := -1, 0
		for x := 0; x < base.Size().Width/8; x++ {
			if base.Bytes()[y*base.Stride()+x] == target.Bytes()[y*target.Stride()+x] {
				continue
			}
			if first < 0 {
				first = x
			}
			last = x
		}
		if first >= 0 {
			bounds = bounds.Union(image.Rect(first*8, y, (last+1)*8, y+1))
		}
	}
	return bounds
}

func expandSpan(start, end, minimum, limit int) (int, int) {
	if end-start >= minimum {
		return start, end
	}
	end = min(limit, start+minimum)
	return end - minimum, end
}

func regionCrop(frame display.Frame, bounds image.Rectangle) []byte {
	stride := bounds.Dx() / 8
	out := make([]byte, stride*bounds.Dy())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		start := y*frame.Stride() + bounds.Min.X/8
		copy(out[(y-bounds.Min.Y)*stride:], frame.Bytes()[start:start+stride])
	}
	return out
}
