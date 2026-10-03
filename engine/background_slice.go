package engine

import (
	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

func (p *painter) backgroundRect(e *element, origin geometry.Rect, maxHeight, width, height float64) (geometry.Rect, error) {
	sizing := origin
	x, y := e.css.Get("background-size-x"), e.css.Get("background-size-y")
	// Horizontally joined fragments retain their own height except when the
	// image's aspect ratio derives its width from that height. Freeze that size
	// using the tallest fragment, then position it in each fragment's own box.
	// https://www.w3.org/TR/css-break-3/#joining-boxes
	if x.Kind == style.KeywordValue || x.Length.Unit == style.Auto && y.Length.Unit != style.Auto {
		sizing.Height = max(sizing.Height, maxHeight)
	}
	tile, err := geometry.BackgroundRect(sizing, width, height, e.css, p.scene.reference(sizing, true))
	if err != nil || sizing.Height == origin.Height {
		return tile, err
	}
	return geometry.BackgroundPosition(origin, tile.Width, tile.Height, e.css, p.scene.reference(origin, true))
}
