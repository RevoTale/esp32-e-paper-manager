// Package geometry computes bounded CSS boxes without fonts, images or hardware.
// Floats are retained until rasterization; device byte alignment belongs later.
package geometry

import (
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

type Reference struct {
	Width, Height                 float64
	ViewportWidth, ViewportHeight float64
	DefiniteHeight                bool
}

type Intrinsic struct{ Minimum, Preferred float64 }

type Edges struct{ Top, Right, Bottom, Left float64 }

func (e Edges) Horizontal() float64 { return e.Left + e.Right }
func (e Edges) Vertical() float64   { return e.Top + e.Bottom }

type Box struct {
	ContentWidth, ContentHeight float64
	Margin, Padding, Border     Edges
	DefiniteHeight              bool
	reference                   Reference
}

func (b Box) OuterWidth() float64 {
	return b.ContentWidth + b.Padding.Horizontal() + b.Border.Horizontal()
}
func (b Box) OuterHeight() float64 {
	return b.ContentHeight + b.Padding.Vertical() + b.Border.Vertical()
}

func (r Reference) metrics(c style.Computed, vertical bool) style.Metrics {
	containing := r.Width
	if vertical {
		containing = r.Height
	}
	return style.Metrics{Containing: containing, Font: c.FontSize(), RootFont: c.RootFont(), ViewportWidth: r.ViewportWidth, ViewportHeight: r.ViewportHeight}
}

func ResolveWidth(c style.Computed, reference Reference, intrinsic Intrinsic) (Box, error) {
	if !nonnegative(reference.Width) || !nonnegative(reference.Height) || !validIntrinsic(intrinsic) {
		return Box{}, style.ErrValue
	}
	b := Box{reference: reference}
	if err := b.resolveEdges(c); err != nil {
		return Box{}, err
	}
	extra := b.Padding.Horizontal() + b.Border.Horizontal()
	automatic := math.Max(0, reference.Width-b.Margin.Horizontal()-extra)
	if c.Keyword("display") == "inline-block" {
		automatic = math.Min(math.Max(intrinsic.Minimum, automatic), intrinsic.Preferred)
	}
	width, _, err := resolveAxis(c, reference, "width", extra, automatic)
	if err != nil {
		return Box{}, err
	}
	b.ContentWidth = width
	b.autoMargins(c)
	if !nonnegative(b.OuterWidth()) || !b.Margin.valid() {
		return Box{}, style.ErrValue
	}
	return b, nil
}

func (b *Box) ResolveHeight(c style.Computed, natural float64) error {
	if !nonnegative(natural) {
		return style.ErrValue
	}
	extra := b.Padding.Vertical() + b.Border.Vertical()
	height, definite, err := resolveAxis(c, b.reference, "height", extra, natural)
	if err != nil {
		return err
	}
	b.ContentHeight, b.DefiniteHeight = height, definite
	if !nonnegative(b.OuterHeight()) {
		return style.ErrValue
	}
	return nil
}

func validIntrinsic(value Intrinsic) bool {
	return nonnegative(value.Minimum) && nonnegative(value.Preferred) && value.Minimum <= value.Preferred
}

func bounded(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Abs(value) <= 32768
}
func nonnegative(value float64) bool { return bounded(value) && value >= 0 }
