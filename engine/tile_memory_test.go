package engine

import (
	"image"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func TestTileListConsumesSharedTemporaryBudget(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 20, Height: 10}, `<div id="tile" style="width:20px;height:10px;background:url(`+splitImage(t)+`) repeat"></div>`)
	e := s.elements[s.document.ByID["tile"]]
	for _, remaining := range []int{0, 31, 64, 32 * 25} {
		p := painter{scene: s, target: image.NewRGBA(image.Rect(0, 0, 20, 10)), temporary: 32*1024*1024 - remaining}
		before := p.temporary
		if err := p.paintBackground(e, e.borderRect(), 0, e.box.OuterHeight(), nil); err == nil {
			t.Fatalf("paint exceeded temporary budget: remaining=%d", remaining)
		}
		if p.temporary != before {
			t.Fatal("failed painting leaked temporary accounting")
		}
	}
}
