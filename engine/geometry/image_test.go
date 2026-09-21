package geometry

import (
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

func TestObjectFitAndPositionUseRemainingSpace(t *testing.T) {
	for _, tc := range []struct {
		fit  string
		want Rect
	}{
		{"contain", Rect{10, 45, 100, 50}}, {"cover", Rect{-40, 20, 200, 100}},
		{"fill", Rect{10, 20, 100, 100}}, {"none", Rect{-40, 20, 200, 100}},
		{"scale-down", Rect{10, 45, 100, 50}},
	} {
		c := boxStyle(t, "object-fit:"+tc.fit)
		got, err := ObjectRect(Rect{10, 20, 100, 100}, 200, 100, c, Reference{})
		if err != nil || got != tc.want {
			t.Fatalf("%s: %+v %v want %+v", tc.fit, got, err, tc.want)
		}
	}
	c := boxStyle(t, "object-fit:none;object-position:50% 0")
	got, err := ObjectRect(Rect{Width: 100, Height: 100}, 20, 20, c, Reference{})
	if err != nil || got.X != 40 {
		t.Fatalf("percent position: %+v %v", got, err)
	}
}

func TestBackgroundSizeAndPosition(t *testing.T) {
	c := boxStyle(t, "background-size:50% auto;background-position:right bottom")
	got, err := BackgroundRect(Rect{Width: 100, Height: 100}, 200, 100, c, Reference{})
	if err != nil || got != (Rect{50, 75, 50, 25}) {
		t.Fatalf("%+v %v", got, err)
	}
	c = boxStyle(t, "background-size:cover;background-position:center")
	got, err = BackgroundRect(Rect{Width: 100, Height: 100}, 200, 100, c, Reference{})
	if err != nil || got != (Rect{-50, 0, 200, 100}) {
		t.Fatalf("cover %+v %v", got, err)
	}
}

func TestRepeatBudgetRejectsSubpixelExplosionBeforeIteration(t *testing.T) {
	clip := Rect{Width: 800, Height: 480}
	for _, tile := range []Rect{{Width: 1e-9, Height: 1}, {Height: 1}, {Width: -1, Height: 1}} {
		if _, err := Tiles(tile, clip, "repeat", 65536); err == nil {
			t.Fatalf("accepted %+v", tile)
		}
	}
	positions, err := Tiles(Rect{X: 2, Y: 2, Width: 4, Height: 4}, Rect{Width: 8, Height: 8}, "repeat", 9)
	if err != nil || len(positions) != 9 || positions[0] != (Rect{-2, -2, 4, 4}) || positions[8] != (Rect{6, 6, 4, 4}) {
		t.Fatalf("repeat alignment: %+v %v", positions, err)
	}
	if _, err := Tiles(Rect{Width: 4, Height: 4}, Rect{Width: 12, Height: 12}, "repeat", 8); err == nil {
		t.Fatal("tile operation bound ignored")
	}
}

func TestImageBoundsAndScaleDown(t *testing.T) {
	c := boxStyle(t, "object-fit:scale-down")
	got, err := ObjectRect(Rect{Width: 100, Height: 100}, 20, 10, c, Reference{})
	if err != nil || got.Width != 20 || got.Height != 10 {
		t.Fatal("scale-down upscaled a small image")
	}
	if _, err := ObjectRect(Rect{Width: 100, Height: 100}, 0, 10, c, Reference{}); err == nil {
		t.Fatal("zero intrinsic image")
	}
	length, err := style.ParseLength("-100%")
	if err != nil {
		t.Fatal(err)
	}
	if value, err := length.Resolve(style.Metrics{Containing: -100}); err != nil || value != 100 {
		t.Fatal("remaining space must permit negative values for oversized images")
	}
}
