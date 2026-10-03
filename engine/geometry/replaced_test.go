package geometry

import "testing"

func TestReplacedImageSizeConstraints(t *testing.T) {
	for _, tc := range []struct {
		css  string
		w, h float64
	}{
		{"", 200, 100}, {"width:100px", 100, 50}, {"height:30px", 60, 30},
		{"width:80px;height:30px", 80, 30}, {"max-width:100px", 100, 50},
		{"max-height:25px", 50, 25}, {"min-width:400px", 400, 200},
		{"min-height:200px", 400, 200}, {"min-width:400px;max-height:50px", 400, 50},
		{"max-width:100px;min-height:200px", 100, 200},
		{"width:100px;height:100px;box-sizing:border-box;padding:10px", 80, 80},
		{"aspect-ratio:1/1", 200, 200}, {"width:0;height:0", 0, 0},
	} {
		box, err := ResolveReplaced(boxStyle(t, tc.css), Reference{Width: 400, Height: 300, DefiniteHeight: true}, 200, 100)
		if err != nil || box.ContentWidth != tc.w || box.ContentHeight != tc.h {
			t.Errorf("%s: %+v %v; want %g x %g", tc.css, box, err, tc.w, tc.h)
		}
	}
}

func TestReplacedRejectsInvalidDimensionsAndConstraints(t *testing.T) {
	for _, css := range []string{"padding:8192em", "width:8192em", "height:8192em", "min-width:2px;max-width:1px", "min-height:2px;max-height:1px", "aspect-ratio:1/8192", "margin:8192em"} {
		if _, err := ResolveReplaced(boxStyle(t, css), Reference{Width: 400, Height: 300, DefiniteHeight: true}, 200, 100); err == nil {
			t.Errorf("accepted %s", css)
		}
	}
	for _, dims := range [][2]float64{{0, 1}, {1, -1}, {1e20, 1}} {
		if _, err := ResolveReplaced(boxStyle(t, ""), Reference{Width: 400}, dims[0], dims[1]); err == nil {
			t.Errorf("accepted intrinsic %v", dims)
		}
	}
}

func TestImageInlineAutoMarginsRemainZero(t *testing.T) {
	box, err := ResolveReplaced(boxStyle(t, "display:inline;margin:0 auto"), Reference{Width: 400}, 200, 100)
	if err != nil || box.Margin.Left != 0 || box.Margin.Right != 0 {
		t.Fatalf("%+v %v", box, err)
	}
}
