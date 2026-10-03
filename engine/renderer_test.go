package engine

import (
	"context"
	"image"
	"image/color"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func rendered(t *testing.T, width, height int, source string) *image.RGBA {
	t.Helper()
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.RenderRGBA(context.Background(), display.Size{Width: width, Height: height}, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return result.Image
}

func pixel(t *testing.T, img *image.RGBA, x, y int, want color.RGBA) {
	t.Helper()
	if got := img.RGBAAt(x, y); got != want {
		t.Fatalf("(%d,%d): %+v, want %+v", x, y, got, want)
	}
}

func TestRenderBlocksRespectViewportAndAlpha(t *testing.T) {
	img := rendered(t, 80, 60, `<div style="height:60px;background:blue"><div style="width:50%;height:30px;background:rgba(255,0,0,0.5)"></div></div>`)
	pixel(t, img, 10, 10, color.RGBA{R: 128, B: 127, A: 255})
	pixel(t, img, 50, 10, color.RGBA{B: 255, A: 255})
	pixel(t, img, 10, 40, color.RGBA{B: 255, A: 255})
}

func TestOpacityAppliesOnceToWholeGroup(t *testing.T) {
	img := rendered(t, 80, 60, `<div style="position:relative;height:60px;background:blue"><div style="opacity:0.5;height:60px;background:red"><div style="height:30px;background:red"></div></div></div>`)
	want := color.RGBA{R: 128, B: 127, A: 255}
	pixel(t, img, 10, 10, want)
	pixel(t, img, 10, 40, want)
}

func TestNestedStackingContextsCannotEscape(t *testing.T) {
	img := rendered(t, 80, 60, `<div style="position:relative;height:60px"><div style="position:absolute;z-index:1;width:80px;height:60px;background:red"><div style="position:absolute;z-index:999;width:80px;height:60px;background:lime"></div></div><div style="position:absolute;z-index:2;width:40px;height:60px;background:blue"></div></div>`)
	pixel(t, img, 10, 10, color.RGBA{B: 255, A: 255})
	pixel(t, img, 60, 10, color.RGBA{G: 255, A: 255})
}

func TestOverflowClipsPositionedDescendantInAncestor(t *testing.T) {
	img := rendered(t, 80, 60, `<div style="overflow:hidden;width:30px;height:30px"><div style="position:relative;left:20px;top:20px;width:40px;height:40px;background:black"></div></div>`)
	pixel(t, img, 25, 25, color.RGBA{A: 255})
	pixel(t, img, 35, 25, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

func TestNativeTextPaintsAtTopLeftAndWraps(t *testing.T) {
	img := rendered(t, 160, 120, `<p style="font-size:24px;line-height:30px">Привіт <b>світе!</b> Ще рядок.</p>`)
	inkTop, inkNext := 0, 0
	for y := 0; y < 120; y++ {
		for x := 0; x < 160; x++ {
			if img.RGBAAt(x, y).R < 128 {
				if y < 30 {
					inkTop++
				} else {
					inkNext++
				}
			}
		}
	}
	if inkTop < 50 || inkNext < 50 {
		t.Fatalf("missing text ink: top=%d following=%d", inkTop, inkNext)
	}
}

func TestRoundedBorderContainsItsCornerArc(t *testing.T) {
	img := rendered(t, 40, 40, `<div style="width:40px;height:40px;box-sizing:border-box;border:2px solid;border-radius:10px"></div>`)
	pixel(t, img, 3, 3, color.RGBA{A: 255})
	pixel(t, img, 0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	pixel(t, img, 8, 8, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}
