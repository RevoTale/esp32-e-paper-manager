package geometry

import (
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

// ResolveReplaced sizes an image before object-fit maps its pixels into the box.
// Width/height:auto preserves intrinsic ratio, including coupled min/max limits.
// https://www.w3.org/TR/CSS22/visudet.html#min-max-widths
func ResolveReplaced(c style.Computed, ref Reference, width, height float64) (Box, error) {
	if !validImageBounds(Rect{Width: ref.Width, Height: ref.Height}, width, height) {
		return Box{}, style.ErrValue
	}
	b := Box{reference: ref, DefiniteHeight: true}
	if err := b.resolveEdges(c); err != nil {
		return Box{}, err
	}
	ratio := width / height
	if c.Get("aspect-ratio").Kind == style.NumberValue {
		ratio = c.Number("aspect-ratio")
		height = width / ratio
	}
	w, h, err := b.replacedDimensions(c, width, height, ratio)
	if err != nil {
		return Box{}, err
	}
	b.ContentWidth, b.ContentHeight = w, h
	b.autoMargins(c)
	if !nonnegative(b.OuterWidth()) || !nonnegative(b.OuterHeight()) || !b.Margin.valid() {
		return Box{}, style.ErrValue
	}
	return b, nil
}

func (b Box) replacedDimensions(c style.Computed, w, h, ratio float64) (float64, float64, error) {
	ex, ey := b.Padding.Horizontal()+b.Border.Horizontal(), b.Padding.Vertical()+b.Border.Vertical()
	height, fixedHeight, err := resolveAxis(c, b.reference, "height", ey, h)
	if err != nil {
		return 0, 0, err
	}
	if fixedHeight {
		w = height * ratio
	}
	width, fixedWidth, err := resolveAxis(c, b.reference, "width", ex, w)
	if err != nil {
		return 0, 0, err
	}
	if !fixedWidth && !fixedHeight {
		return b.coupledConstraints(c, w, h)
	}
	if !fixedHeight {
		height, _, err = resolveAxis(c, b.reference, "height", ey, width/ratio)
	}
	return width, height, err
}

func (b Box) coupledConstraints(c style.Computed, w, h float64) (float64, float64, error) {
	minW, maxW, err := axisConstraints(c, b.reference, "width", b.Padding.Horizontal()+b.Border.Horizontal())
	if err != nil {
		return 0, 0, err
	}
	minH, maxH, err := axisConstraints(c, b.reference, "height", b.Padding.Vertical()+b.Border.Vertical())
	if err != nil {
		return 0, 0, err
	}
	minimum, maximum := math.Max(minW/w, minH/h), math.Min(maxW/w, maxH/h)
	if minimum <= maximum {
		ratio := math.Max(minimum, math.Min(1, maximum))
		return w * ratio, h * ratio, nil
	}
	// Opposing constraints cannot preserve the ratio; CSS resolves each violated
	// edge. object-fit can preserve the source ratio inside that constrained box.
	if minW/w > maxH/h {
		return minW, maxH, nil
	}
	return maxW, minH, nil
}
