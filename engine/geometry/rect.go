package geometry

import "math"

// Rect is a top-left-origin logical rectangle; Width/Height never imply an
// allocation. Raster bounds must be intersected with the physical viewport.
type Rect struct{ X, Y, Width, Height float64 }

func (r Rect) Right() float64  { return r.X + r.Width }
func (r Rect) Bottom() float64 { return r.Y + r.Height }
func (r Rect) Empty() bool     { return r.Width == 0 || r.Height == 0 }

func (r Rect) Valid() bool {
	return bounded(r.X) && bounded(r.Y) && nonnegative(r.Width) && nonnegative(r.Height) && bounded(r.Right()) && bounded(r.Bottom())
}

func (r Rect) Intersect(other Rect) Rect {
	x, y := math.Max(r.X, other.X), math.Max(r.Y, other.Y)
	return Rect{x, y, math.Max(0, math.Min(r.Right(), other.Right())-x), math.Max(0, math.Min(r.Bottom(), other.Bottom())-y)}
}
