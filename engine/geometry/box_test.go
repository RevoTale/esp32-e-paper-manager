package geometry

import (
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

func boxStyle(t *testing.T, source string) style.Computed {
	t.Helper()
	block, err := style.Parse(source, 1)
	if err != nil {
		t.Fatal(err)
	}
	computed, err := style.Compute("div", block, nil, style.Metrics{ViewportWidth: 800, ViewportHeight: 480})
	if err != nil {
		t.Fatal(err)
	}
	return computed
}

func TestBoxSizingAndAutoMargins(t *testing.T) {
	for _, tc := range []struct {
		css                  string
		content, outer, left float64
	}{
		{"width:200px;padding:10px;border:2px solid; margin:0 auto", 200, 224, 88},
		{"width:200px;padding:10px;border:2px solid; margin:0 auto;box-sizing:border-box", 176, 200, 100},
		{"width:50%;padding:10%;box-sizing:border-box", 120, 200, 0},
		{"padding:10px;border:2px solid", 376, 400, 0},
		{"max-width:100px;margin-left:auto", 100, 100, 300},
	} {
		computed := boxStyle(t, tc.css)
		box, err := ResolveWidth(computed, Reference{Width: 400, Height: 200, DefiniteHeight: true, ViewportWidth: 800, ViewportHeight: 480}, Intrinsic{})
		if err != nil || box.ContentWidth != tc.content || box.OuterWidth() != tc.outer || box.Margin.Left != tc.left {
			t.Fatalf("%s: %+v %v; want content=%v outer=%v left=%v", tc.css, box, err, tc.content, tc.outer, tc.left)
		}
	}
}

func TestHeightPercentageAndMinMax(t *testing.T) {
	computed := boxStyle(t, "height:50%;min-height:10px;max-height:120px;padding:2px;box-sizing:border-box")
	for _, tc := range []struct {
		definite bool
		want     float64
	}{{true, 96}, {false, 20}} {
		box, err := ResolveWidth(computed, Reference{Width: 400, Height: 200, DefiniteHeight: tc.definite}, Intrinsic{})
		if err != nil {
			t.Fatal(err)
		}
		if err = box.ResolveHeight(computed, 20); err != nil || box.ContentHeight != tc.want || box.DefiniteHeight != tc.definite {
			t.Fatalf("%+v %v", box, err)
		}
	}
	if _, err := ResolveWidth(boxStyle(t, "min-width:200px;max-width:100px"), Reference{Width: 400}, Intrinsic{}); err == nil {
		t.Fatal("incompatible dimensions accepted")
	}
}

func TestInlineBlockShrinkToFitAndSignedMargins(t *testing.T) {
	computed := boxStyle(t, "display:inline-block;padding:10px;margin-left:-5px")
	box, err := ResolveWidth(computed, Reference{Width: 150}, Intrinsic{Minimum: 30, Preferred: 200})
	if err != nil || box.ContentWidth != 135 || box.Margin.Left != -5 {
		t.Fatalf("%+v %v", box, err)
	}
}

func TestGeometryRejectsOverflowWithoutAllocating(t *testing.T) {
	computed := boxStyle(t, "padding:8192em")
	if _, err := ResolveWidth(computed, Reference{Width: 400}, Intrinsic{}); err == nil {
		t.Fatal("unbounded resolved padding")
	}
	for _, ref := range []Reference{{Width: -1}, {Width: 1e100}, {Height: -1}} {
		if _, err := ResolveWidth(boxStyle(t, ""), ref, Intrinsic{}); err == nil {
			t.Fatal("invalid containing block")
		}
	}
}
