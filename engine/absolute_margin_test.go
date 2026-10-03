package engine

import (
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
)

// Independent solutions of CSS2.2 10.3.7/10.6.4 margin-box equations.
func TestAbsolutePairedInsetsResolveAutomaticMargins(t *testing.T) {
	for _, tc := range []struct {
		style, dir string
		x, y       float64
	}{
		{"margin:auto", "ltr", 35, 30},
		{"margin-left:auto", "ltr", 60, 10},
		{"margin-right:auto", "ltr", 10, 10},
		{"margin-top:auto", "ltr", 10, 50},
		{"margin-bottom:auto", "ltr", 10, 10},
		{"margin:auto;width:120px", "ltr", 10, 30},
		{"margin:auto;width:120px", "rtl", -30, 30},
		{"margin:auto;height:100px", "ltr", 35, -10},
	} {
		s := layoutDocument(t, display.Size{Width: 100, Height: 80}, `<div dir="`+tc.dir+`" style="position:relative;height:80px"><div id="a" style="position:absolute;left:10px;right:10px;top:10px;bottom:10px;width:30px;height:20px;`+tc.style+`"></div></div>`)
		e := s.elements[s.document.ByID["a"]]
		if e.x != tc.x || e.y != tc.y {
			t.Fatalf("%s %s: (%g,%g), want (%g,%g)", tc.dir, tc.style, e.x, e.y, tc.x, tc.y)
		}
	}
}

func TestAbsoluteRTLOverconstraintUsesContainingDirection(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 100, Height: 80}, `<div dir="rtl" style="position:relative;height:80px"><div id="a" dir="ltr" style="position:absolute;left:10px;right:20px;top:0;width:30px;height:20px"></div></div>`)
	expectBox(t, s, "a", geometry.Rect{X: 50, Width: 30, Height: 20})
}
