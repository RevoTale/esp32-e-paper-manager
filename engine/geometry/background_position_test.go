package geometry

import "testing"

func TestBackgroundPositionPreservesSharedFragmentSize(t *testing.T) {
	c := boxStyle(t, "background-position:right bottom")
	got, err := BackgroundPosition(Rect{10, 20, 100, 30}, 60, 40, c, Reference{})
	if err != nil || got != (Rect{50, 10, 60, 40}) {
		t.Fatalf("position %+v, %v", got, err)
	}
	for _, area := range []Rect{{Width: -1, Height: 20}, {Width: 40000, Height: 20}} {
		if _, err := BackgroundPosition(area, 60, 40, c, Reference{}); err == nil {
			t.Fatalf("accepted invalid fragment %+v", area)
		}
	}
	if _, err := BackgroundPosition(Rect{Width: 20, Height: 20}, -1, 40, c, Reference{}); err == nil {
		t.Fatal("accepted negative shared image size")
	}
	got, err = BackgroundPosition(Rect{Width: 20, Height: 20}, 0, 40, c, Reference{})
	if err != nil || !got.Empty() {
		t.Fatalf("zero-sized background must remain unpainted: %+v, %v", got, err)
	}
}
