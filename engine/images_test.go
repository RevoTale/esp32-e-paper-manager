package engine

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func splitImage(t *testing.T) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			c := color.NRGBA{R: 255, A: 255}
			if x >= 2 {
				c = color.NRGBA{B: 255, A: 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func TestImageIntrinsicAndHTMLDimensionHints(t *testing.T) {
	url := splitImage(t)
	for _, tc := range []struct {
		attrs string
		w, h  float64
	}{
		{"", 4, 2}, {`width="40"`, 40, 20},
		{`width="40" style="width:20px"`, 20, 10},
		{`style="height:30px"`, 60, 30},
		{`style="max-width:2px"`, 2, 1},
		{`style="max-height:1px"`, 2, 1},
	} {
		s := layoutDocument(t, display.Size{Width: 100, Height: 100}, `<img id="img" src="`+url+`" `+tc.attrs+`>`)
		e := s.elements[s.document.ByID["img"]]
		if e.box.ContentWidth != tc.w || e.box.ContentHeight != tc.h {
			t.Errorf("%s: size %g x %g, want %g x %g", tc.attrs, e.box.ContentWidth, e.box.ContentHeight, tc.w, tc.h)
		}
	}
}

func TestImageObjectFitContainsBeforeClipping(t *testing.T) {
	img := rendered(t, 80, 80, `<img style="display:block;width:80px;height:80px;object-fit:contain;background:lime" src="`+splitImage(t)+`">`)
	pixel(t, img, 10, 10, color.RGBA{G: 255, A: 255})
	pixel(t, img, 10, 30, color.RGBA{R: 255, A: 255})
	pixel(t, img, 70, 30, color.RGBA{B: 255, A: 255})
	pixel(t, img, 10, 70, color.RGBA{G: 255, A: 255})
}

func TestBackgroundRepeatAndRelativePosition(t *testing.T) {
	url := splitImage(t)
	img := rendered(t, 20, 10, `<div style="width:20px;height:10px;background-image:url(`+url+`);background-repeat:repeat;background-size:4px 2px"></div>`)
	for _, x := range []int{0, 4, 8, 16} {
		pixel(t, img, x, 8, color.RGBA{R: 255, A: 255})
		pixel(t, img, x+3, 8, color.RGBA{B: 255, A: 255})
	}
	img = rendered(t, 20, 10, `<div style="width:20px;height:10px;background:url(`+url+`) right bottom / 4px 2px no-repeat"></div>`)
	pixel(t, img, 16, 8, color.RGBA{R: 255, A: 255})
	pixel(t, img, 19, 9, color.RGBA{B: 255, A: 255})
	pixel(t, img, 0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}
