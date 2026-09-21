package engine

import (
	"image/color"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func TestInlineBackgroundContinuesAcrossLineBreak(t *testing.T) {
	source := `<div style="padding:10px;font-size:20px;line-height:40px"><span id="s" style="color:transparent;background:url(` + splitImage(t) + `) left top / 100% 100% no-repeat">aaaa<br>aaaa</span></div>`
	s := layoutDocument(t, display.Size{Width: 180, Height: 120}, source)
	fragments := elementID(s, "s").fragments
	if len(fragments) != 2 {
		t.Fatalf("want two fragments, got %+v", fragments)
	}
	img := rendered(t, 180, 120, source)
	pixel(t, img, 15, int(fragments[0].Y)+5, color.RGBA{R: 255, A: 255})
	pixel(t, img, 40, int(fragments[1].Y)+5, color.RGBA{B: 255, A: 255})
}

func TestInlineBackgroundAspectRatioUsesTallestFragment(t *testing.T) {
	source := `<div style="padding:10px;font-size:20px;line-height:55px"><span id="s" style="color:transparent;background:url(` + splitImage(t) + `) left top / auto 100% no-repeat">aaaaaaaaaaaa<br><span style="font-size:40px">a</span></span></div>`
	s := layoutDocument(t, display.Size{Width: 220, Height: 160}, source)
	fragments := elementID(s, "s").fragments
	if len(fragments) != 2 || fragments[1].Height <= fragments[0].Height {
		t.Fatalf("want unequal-height fragments, got %+v", fragments)
	}
	img := rendered(t, 220, 160, source)
	pixel(t, img, 85, int(fragments[0].Y)+5, color.RGBA{B: 255, A: 255})
}

func TestInlineZeroSizedNoRepeatBackgroundStaysUnpainted(t *testing.T) {
	source := `<div style="padding:10px;font-size:20px;line-height:55px"><span style="color:transparent;background:url(` + splitImage(t) + `) left top / auto 0px no-repeat">aaaaaaaaaaaa<br><span style="font-size:40px">a</span></span></div>`
	img := rendered(t, 220, 160, source)
	pixel(t, img, 85, 30, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}
