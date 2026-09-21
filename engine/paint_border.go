package engine

import (
	"image/color"
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
	"github.com/tdewolff/canvas"
)

// Borders form a ring, not four clipped strips: strips omit rounded corner
// arcs. Inner radii subtract the corresponding edge independently.
// https://www.w3.org/TR/css-backgrounds-3/#corner-shaping
func (p *painter) borderRing(e *element, area geometry.Rect, radius float64, clips []clip) error {
	b := e.box.Border
	if area.Empty() || b.Horizontal()+b.Vertical() == 0 {
		return nil
	}
	ring := canvas.RoundedRectangle(area.Width, area.Height, radius)
	w, h := area.Width-b.Horizontal(), area.Height-b.Vertical()
	if w > 0 && h > 0 {
		ring = ring.Append(innerBorder(w, h, radius, b).Translate(b.Left, b.Top))
	}
	for i, side := range []string{"top", "right", "bottom", "left"} {
		style := canvas.DefaultStyle
		style.FillRule = canvas.EvenOdd
		style.Fill = canvas.Paint{Color: color.RGBAModel.Convert(e.css.Color("border-" + side + "-color")).(color.RGBA)}
		partition := clip{area: area, border: b, side: i + 1}
		if err := p.drawPath(ring, style, canvas.Identity.Translate(area.X, area.Y), append(clips, partition)); err != nil {
			return err
		}
	}
	return nil
}

func innerBorder(w, h, radius float64, b geometry.Edges) *canvas.Path {
	left, right := math.Max(0, radius-b.Left), math.Max(0, radius-b.Right)
	top, bottom := math.Max(0, radius-b.Top), math.Max(0, radius-b.Bottom)
	path := &canvas.Path{}
	path.MoveTo(0, top)
	path.ArcTo(left, top, 0, false, true, left, 0)
	path.LineTo(w-right, 0)
	path.ArcTo(right, top, 0, false, true, w, top)
	path.LineTo(w, h-bottom)
	path.ArcTo(right, bottom, 0, false, true, w-right, h)
	path.LineTo(left, h)
	path.ArcTo(left, bottom, 0, false, true, 0, h-bottom)
	path.Close()
	return path
}
