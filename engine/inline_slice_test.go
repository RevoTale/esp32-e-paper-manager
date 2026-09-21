package engine

import (
	"image/color"
	"math"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func TestWrappedInlineBordersAppearOnlyAtUnbrokenSides(t *testing.T) {
	for _, direction := range []string{"ltr", "rtl"} {
		t.Run(direction, func(t *testing.T) {
			source := `<div dir="` + direction + `" style="padding:10px;font-size:20px;line-height:36px"><span id="s" style="color:transparent;background:lime;border:3px solid red">aa<br>bb<br>cc</span></div>`
			s := layoutDocument(t, display.Size{Width: 160, Height: 140}, source)
			fragments := elementID(s, "s").fragments
			if len(fragments) != 3 {
				t.Fatalf("want three line fragments, got %+v", fragments)
			}
			img := rendered(t, 160, 140, source)
			for i, area := range fragments {
				left, right := color.RGBA{G: 255, A: 255}, color.RGBA{G: 255, A: 255}
				if i == 0 {
					left = color.RGBA{R: 255, A: 255}
				}
				if i == len(fragments)-1 {
					right = color.RGBA{R: 255, A: 255}
				}
				if direction == "rtl" {
					left, right = right, left
				}
				y := int(area.Y + area.Height/2)
				pixel(t, img, int(math.Ceil(area.X))+1, y, left)
				pixel(t, img, int(math.Floor(area.Right()))-2, y, right)
				pixel(t, img, int(area.X+area.Width/2), int(math.Ceil(area.Y))+1, color.RGBA{R: 255, A: 255})
			}
		})
	}
}

func TestWrappedInlineRadiusDoesNotRoundBrokenEdges(t *testing.T) {
	source := `<div style="padding:10px;font-size:20px;line-height:40px"><span id="s" style="color:transparent;background:lime;border:3px solid red;border-radius:12px">aaaa<br>bbbb<br>cccc</span></div>`
	s := layoutDocument(t, display.Size{Width: 180, Height: 160}, source)
	fragments := elementID(s, "s").fragments
	if len(fragments) != 3 {
		t.Fatalf("want three fragments, got %+v", fragments)
	}
	img := rendered(t, 180, 160, source)
	middle := fragments[1]
	pixel(t, img, int(math.Ceil(middle.X))+1, int(math.Ceil(middle.Y))+1, color.RGBA{R: 255, A: 255})
	pixel(t, img, int(math.Floor(middle.Right()))-2, int(math.Ceil(middle.Y))+1, color.RGBA{R: 255, A: 255})
}

func TestRTLFragmentAdvancesUseThePhysicalUnbrokenEdges(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 180, Height: 160}, `<div dir="rtl" style="font-size:20px;line-height:40px"><span id="s" style="border-left:2px solid red;border-right:6px solid blue;padding-left:3px;padding-right:7px">aa<br>aa</span></div>`)
	e := elementID(s, "s")
	if len(e.fragments) != 2 {
		t.Fatalf("want two fragments, got %+v", e.fragments)
	}
	textWidth := s.fonts.face(e.css).TextWidth("aa")
	if math.Abs(e.fragments[0].Width-textWidth-13) > 1e-9 || math.Abs(e.fragments[1].Width-textWidth-5) > 1e-9 {
		t.Fatalf("physical right edge must advance first line and left edge last: %+v", e.fragments)
	}
}
