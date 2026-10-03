package engine

import (
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
)

func TestRelativePercentHeightDoesNotUseContentDependentHeight(t *testing.T) {
	for _, tc := range []struct {
		parent, offsets string
		y               float64
	}{
		{"", "top:50%", 0}, {"", "bottom:50%", 0}, {"", "top:50%;bottom:5px", -5},
		{"height:80px", "top:50%", 40}, {"height:80px", "bottom:50%", -40},
		{"", "top:5px;bottom:50%", 5},
	} {
		s := layoutDocument(t, display.Size{Width: 100, Height: 100}, `<div style="`+tc.parent+`"><div id="a" style="position:relative;height:20px;`+tc.offsets+`"></div><div id="b" style="height:10px"></div></div>`)
		expectBox(t, s, "a", geometry.Rect{X: 0, Y: tc.y, Width: 100, Height: 20})
		expectBox(t, s, "b", geometry.Rect{Y: 20, Width: 100, Height: 10})
	}
}

func TestRelativeHorizontalOverconstraintRespectsDirection(t *testing.T) {
	for _, tc := range []struct {
		dir string
		x   float64
	}{{"ltr", 10}, {"rtl", -20}} {
		s := layoutDocument(t, display.Size{Width: 100, Height: 100}, `<div dir="`+tc.dir+`"><div id="a" dir="ltr" style="position:relative;left:10px;right:20px;width:100px;height:20px"></div></div>`)
		expectBox(t, s, "a", geometry.Rect{X: tc.x, Width: 100, Height: 20})
	}
}
