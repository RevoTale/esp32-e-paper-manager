package engine

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
)

func bitmapURL(t *testing.T, pixels []color.NRGBA, width int) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, len(pixels)/width))
	for i, c := range pixels {
		img.SetNRGBA(i%width, i/width, c)
	}
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data.Bytes())
}

// Active replacements for all three ignored Blitz bugs. These retain the
// original geometry, changing asset injection to canonical PNG. Native CPU
// bilinear sampling/Bayer quantization has its own explicit pixel oracle.
// Historical evidence: docs/blitz-image-layout-bugs.md, blitz-positioning-gap.md.
func TestHistoricalImageIntrinsicCollapse(t *testing.T) {
	black := color.NRGBA{A: 255}
	url := bitmapURL(t, []color.NRGBA{black, black, black, black}, 2)
	assertMono(t, display.Size{Width: 8, Height: 8}, `<img src="`+url+`" style="position:absolute;left:2px;top:1px">`, []byte{0, 0x30, 0x30, 0, 0, 0, 0, 0})
}

func TestHistoricalImageContainClipping(t *testing.T) {
	url := bitmapURL(t, []color.NRGBA{{A: 255}, {R: 255, G: 255, B: 255, A: 255}}, 2)
	source := `<div style="width:3px;height:3px;overflow:hidden"><img id="image" src="` + url + `" style="display:block;width:4px;height:4px;object-fit:contain;object-position:center"></div>`
	s := layoutDocument(t, display.Size{Width: 8, Height: 8}, source)
	expectBox(t, s, "image", geometry.Rect{Width: 4, Height: 4})
	img := rendered(t, 8, 8, source)
	// The original worker oracle assumed a hard black/white sampling edge.
	// Our specified bilinear edge has gray samples before the anchored dither;
	// neither interpolation nor clipping is allowed to shrink the 4x4 box.
	pixel(t, img, 1, 1, color.RGBA{R: 255 / 4, G: 255 / 4, B: 255 / 4, A: 255})
	pixel(t, img, 2, 1, color.RGBA{R: 191, G: 191, B: 191, A: 255})
	assertMono(t, display.Size{Width: 8, Height: 8}, source, []byte{0, 0xe0, 0xc0, 0, 0, 0, 0, 0})
}

func TestHistoricalStaticWrapperDoesNotCaptureAbsolute(t *testing.T) {
	expected := make([]byte, 3*16)
	expected[12*3+2], expected[13*3+2] = 0x0c, 0x0c
	assertMono(t, display.Size{Width: 24, Height: 16}, `<div style="position:absolute;left:2px;top:2px;width:20px;height:12px"><div style="position:static;width:8px;height:6px"><div style="position:absolute;right:0;bottom:0;width:2px;height:2px;background:black"></div></div></div>`, expected)
}

func assertMono(t *testing.T, size display.Size, source string, expected []byte) {
	t.Helper()
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	frame, err := r.Render(context.Background(), size, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(frame.Bytes(), expected) {
		t.Fatalf("mono %x, expected %x", frame.Bytes(), expected)
	}
}
