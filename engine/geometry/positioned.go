package geometry

import (
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

// ResolveAbsoluteWidth keeps percentage bases at the containing padding box,
// but auto width uses the space between insets or CSS shrink-to-fit.
// https://www.w3.org/TR/CSS22/visudet.html#abs-non-replaced-width
func ResolveAbsoluteWidth(c style.Computed, ref Reference, intrinsic Intrinsic) (Box, error) {
	b := Box{reference: ref}
	if !validIntrinsic(intrinsic) {
		return Box{}, style.ErrValue
	}
	if err := b.resolveEdges(c); err != nil {
		return Box{}, err
	}
	extra := b.Padding.Horizontal() + b.Border.Horizontal()
	available, stretch, err := insetSpace(c, ref, "left", "right", ref.Width-extra-b.Margin.Horizontal(), false)
	if err != nil {
		return Box{}, err
	}
	if !stretch {
		available = math.Min(math.Max(intrinsic.Minimum, available), intrinsic.Preferred)
	}
	b.ContentWidth, _, err = resolveAxis(c, ref, "width", extra, math.Max(0, available))
	if err != nil || !nonnegative(b.OuterWidth()) {
		return Box{}, style.ErrValue
	}
	return b, nil
}

func (b *Box) ResolveAbsoluteHeight(c style.Computed, natural float64) error {
	extra := b.Padding.Vertical() + b.Border.Vertical()
	available, stretch, err := insetSpace(c, b.reference, "top", "bottom", b.reference.Height-extra-b.Margin.Vertical(), true)
	if err != nil {
		return err
	}
	if stretch {
		natural = math.Max(0, available)
	}
	if err := b.ResolveHeight(c, natural); err != nil {
		return err
	}
	b.DefiniteHeight = b.DefiniteHeight || stretch
	return nil
}

func insetSpace(c style.Computed, ref Reference, start, end string, available float64, vertical bool) (float64, bool, error) {
	metrics := ref.metrics(c, vertical)
	count := 0
	for _, name := range []string{start, end} {
		if c.Length(name).Unit == style.Auto {
			continue
		}
		value, err := c.Length(name).Resolve(metrics)
		if err != nil {
			return 0, false, err
		}
		available -= value
		count++
	}
	return available, count == 2, nil
}
