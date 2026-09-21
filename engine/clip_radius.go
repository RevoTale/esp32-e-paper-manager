package engine

import (
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
)

func insetRect(area geometry.Rect, inset geometry.Edges) geometry.Rect {
	return geometry.Rect{X: area.X + inset.Left, Y: area.Y + inset.Top, Width: math.Max(0, area.Width-inset.Horizontal()), Height: math.Max(0, area.Height-inset.Vertical())}
}

func (p *painter) innerClip(e *element, inset geometry.Edges) (clip, error) {
	area := e.borderRect()
	radius, err := p.radius(e, area)
	return clip{area: insetRect(area, inset), radius: radius, inset: inset}, err
}

// Inner radii subtract the respective borders/padding; unequal edges produce
// elliptical, not circular, inner corners. Replaced content uses the content edge.
// https://www.w3.org/TR/css-backgrounds-3/#corner-clipping
func (c clip) roundedContains(x, y float64) bool {
	a := c.area
	left, right := math.Max(0, c.radius-c.inset.Left), math.Max(0, c.radius-c.inset.Right)
	top, bottom := math.Max(0, c.radius-c.inset.Top), math.Max(0, c.radius-c.inset.Bottom)
	for _, corner := range [][4]float64{
		{a.X + left, a.Y + top, -left, -top},
		{a.Right() - right, a.Y + top, right, -top},
		{a.X + left, a.Bottom() - bottom, -left, bottom},
		{a.Right() - right, a.Bottom() - bottom, right, bottom},
	} {
		rx, ry := corner[2], corner[3]
		if rx == 0 || ry == 0 {
			continue
		}
		dx, dy := (x-corner[0])/rx, (y-corner[1])/ry
		if dx > 0 && dy > 0 && dx*dx+dy*dy > 1 {
			return false
		}
	}
	return true
}
