package engine

import (
	"image/color"
	"testing"
)

func TestSquareBorderAlphaCompositesCornerOnce(t *testing.T) {
	img := rendered(t, 40, 40, `<div style="width:40px;height:40px;box-sizing:border-box;border:6px solid rgba(255,0,0,0.5)"></div>`)
	want := color.RGBA{R: 255, G: 127, B: 127, A: 255}
	pixel(t, img, 2, 2, want)
	pixel(t, img, 20, 2, want)
}

func TestImageRespectsOwnRoundedContentClip(t *testing.T) {
	img := rendered(t, 40, 40, `<img style="display:block;width:40px;height:40px;border-radius:15px" src="`+splitImage(t)+`">`)
	pixel(t, img, 0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	pixel(t, img, 5, 20, color.RGBA{R: 255, A: 255})
}

func TestBackgroundPositionOriginIsPaddingBox(t *testing.T) {
	img := rendered(t, 30, 30, `<div style="width:20px;height:20px;border:5px solid transparent;background:url(`+splitImage(t)+`) left top / 4px 2px no-repeat"></div>`)
	pixel(t, img, 0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	pixel(t, img, 5, 5, color.RGBA{R: 255, A: 255})
	pixel(t, img, 8, 5, color.RGBA{B: 255, A: 255})
}
