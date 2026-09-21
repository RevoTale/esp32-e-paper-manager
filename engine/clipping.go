package engine

import (
	"image"
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

type clip struct {
	area   geometry.Rect
	radius float64
	border geometry.Edges
	inset  geometry.Edges
	side   int
}

func (p *painter) clips(e *element) ([]clip, error) {
	result := []clip{{area: p.scene.viewport}}
	escaping := false
	for child, parent := e, e.parent; parent != nil; child, parent = parent, parent.parent {
		if child.css.Keyword("position") == "absolute" {
			escaping = true
		}
		if parent.css.Keyword("position") != "static" {
			escaping = false
		}
		if !escaping && clipsOverflow(parent) {
			inner, err := p.innerClip(parent, parent.box.Border)
			if err != nil {
				return nil, err
			}
			result = append(result, inner)
		}
	}
	return result, nil
}

// Overflow does not clip non-replaced inline boxes. Absolutely positioned
// descendants skip clips between themselves and their containing block.
// https://www.w3.org/TR/CSS22/visufx.html#overflow
func clipsOverflow(e *element) bool {
	return e.css.Keyword("display") != "inline" && e.css.Keyword("overflow") != "visible"
}

func (p *painter) radius(e *element, area geometry.Rect) (float64, error) {
	r, err := e.css.Length("border-radius").Resolve(style.Metrics{Containing: math.Min(area.Width, area.Height), Font: e.css.FontSize(), RootFont: e.css.RootFont(), ViewportWidth: p.scene.viewport.Width, ViewportHeight: p.scene.viewport.Height})
	if err != nil {
		return 0, e.reject("border-radius")
	}
	return math.Min(r, math.Min(area.Width, area.Height)/2), nil
}

func pixelBounds(area geometry.Rect, clips []clip) image.Rectangle {
	for _, c := range clips {
		area = area.Intersect(c.area)
	}
	if area.Empty() {
		return image.Rectangle{}
	}
	return image.Rect(int(math.Floor(area.X)), int(math.Floor(area.Y)), int(math.Ceil(area.Right())), int(math.Ceil(area.Bottom())))
}

func (c clip) contains(x, y float64) bool {
	a := c.area
	if x < a.X || y < a.Y || x >= a.Right() || y >= a.Bottom() {
		return false
	}
	if c.side != 0 && c.borderSide(x, y) != c.side {
		return false
	}
	if c.radius == 0 {
		return true
	}
	return c.roundedContains(x, y)
}

func (c clip) borderSide(x, y float64) int {
	nearest, selected := math.Inf(1), 0
	for i, edge := range [][2]float64{{y - c.area.Y, c.border.Top}, {c.area.Right() - x, c.border.Right}, {c.area.Bottom() - y, c.border.Bottom}, {x - c.area.X, c.border.Left}} {
		if edge[1] > 0 && edge[0]/edge[1] < nearest {
			nearest, selected = edge[0]/edge[1], i+1
		}
	}
	return selected
}

func insideClips(clips []clip, x, y float64) bool {
	for _, c := range clips {
		if !c.contains(x, y) {
			return false
		}
	}
	return true
}
