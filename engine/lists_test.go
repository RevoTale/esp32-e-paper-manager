package engine

import (
	"github.com/RevoTale/esp32-e-paper-manager/display"
	"testing"
)

func TestListMarkersNumberEachListIndependently(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 200, Height: 200}, `<ol><li id="one">one</li><li style="display:none">hidden</li><li id="two">two<ul><li id="bullet">nested</li></ul><ol><li id="nested">again</li></ol></li><li id="empty"></li></ol>`)
	for id, want := range map[string]string{"one": "1.", "two": "2.", "bullet": "•", "nested": "1.", "empty": "3."} {
		e := s.elements[s.document.ByID[id]]
		if got := listMarker(e); got != want {
			t.Errorf("%s: %q, want %q", id, got, want)
		}
		if len(e.glyphs) == 0 || e.box.ContentHeight == 0 {
			t.Errorf("%s has no marker/line box", id)
		}
	}
}

func TestListMarkerPaintsInsideDefaultIndent(t *testing.T) {
	img := rendered(t, 100, 60, `<ul><li>hello</li></ul>`)
	ink := 0
	for y := 0; y < 25; y++ {
		for x := 0; x < 24; x++ {
			if img.RGBAAt(x, y).R < 128 {
				ink++
			}
		}
	}
	if ink == 0 {
		t.Fatal("bullet missing from indent")
	}
}
