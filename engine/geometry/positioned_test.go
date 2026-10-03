package geometry

import "testing"

func TestPositionedSizingStretchAndShrink(t *testing.T) {
	for _, tc := range []struct {
		css  string
		w, h float64
	}{
		{"left:10px;right:20px;top:5px;bottom:15px", 170, 80},
		{"left:10px", 80, 30}, {"right:190px", 40, 30},
		{"left:10%;right:10%;padding:5px;border:1px solid", 148, 30},
		{"left:10px;right:10px;width:50px;top:0;bottom:0;height:20px", 50, 20},
		{"left:300px;right:300px;top:300px;bottom:300px", 0, 0},
	} {
		c := boxStyle(t, "position:absolute;"+tc.css)
		box, err := ResolveAbsoluteWidth(c, Reference{Width: 200, Height: 100, DefiniteHeight: true}, Intrinsic{40, 80})
		if err != nil {
			t.Fatal(err)
		}
		err = box.ResolveAbsoluteHeight(c, 30)
		if err != nil || box.ContentWidth != tc.w || box.ContentHeight != tc.h {
			t.Errorf("%s: %+v %v", tc.css, box, err)
		}
	}
}

func TestPositionedRejectsInvalidOperands(t *testing.T) {
	for _, css := range []string{"padding:8192em", "left:8192em", "right:8192em", "width:8192em"} {
		if _, err := ResolveAbsoluteWidth(boxStyle(t, css), Reference{Width: 200}, Intrinsic{40, 80}); err == nil {
			t.Errorf("accepted %s", css)
		}
	}
	if _, err := ResolveAbsoluteWidth(boxStyle(t, ""), Reference{Width: 200}, Intrinsic{80, 40}); err == nil {
		t.Fatal("inverted intrinsic range")
	}
	for _, css := range []string{"top:8192em", "bottom:8192em", "height:8192em"} {
		c := boxStyle(t, css)
		box, err := ResolveAbsoluteWidth(c, Reference{Width: 200}, Intrinsic{40, 80})
		if err != nil {
			t.Fatal(err)
		}
		if err := box.ResolveAbsoluteHeight(c, 10); err == nil {
			t.Errorf("accepted %s", css)
		}
	}
}
