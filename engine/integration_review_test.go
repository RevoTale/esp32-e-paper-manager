package engine

import (
	"bytes"
	"image/color"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
)

func TestNonReplacedInlineOverflowDoesNotEraseText(t *testing.T) {
	want := rendered(t, 120, 60, `<span>AA<b>BB</b></span>`)
	for _, overflow := range []string{"hidden", "clip"} {
		got := rendered(t, 120, 60, `<span style="overflow:`+overflow+`">AA<b>BB</b></span>`)
		if !bytes.Equal(got.Pix, want.Pix) {
			t.Fatalf("non-replaced inline overflow:%s changed pixels", overflow)
		}
	}
}

func TestAtomicInlineObjectContributesToOwnerFragment(t *testing.T) {
	const source = `<span id="owner" style="background:red"><span style="display:inline-block;width:20px;height:20px"></span></span>`
	s := layoutDocument(t, display.Size{Width: 120, Height: 60}, source)
	e := elementID(s, "owner")
	if len(e.fragments) != 1 || e.fragments[0].Width != 20 {
		t.Fatalf("atomic object missing from owner geometry: %+v", e.fragments)
	}
	img := rendered(t, 120, 60, source)
	pixel(t, img, 10, 10, color.RGBA{R: 255, A: 255})
}

func TestAbsoluteEscapesInterveningOverflowNotContainingBlock(t *testing.T) {
	const source = `<div style="position:relative;width:50px;height:40px;overflow:hidden"><div style="width:20px;height:20px;overflow:hidden"><div style="position:absolute;left:25px;top:0;width:40px;height:20px;background:red"><span style="display:inline-block;width:40px;height:10px;background:blue"></span></div></div></div>`
	img := rendered(t, 100, 60, source)
	pixel(t, img, 30, 18, color.RGBA{R: 255, A: 255})
	pixel(t, img, 30, 7, color.RGBA{B: 255, A: 255})
	pixel(t, img, 55, 7, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

func TestInlineAbsoluteEstimateUsesCurrentParagraphOrigin(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 120, Height: 80}, `<div style="padding:20px"><div style="height:10px"></div><span>text<span id="a" style="position:absolute;width:5px;height:5px"></span></span></div>`)
	expectBox(t, s, "a", geometry.Rect{X: 20, Y: 30, Width: 5, Height: 5})
}
