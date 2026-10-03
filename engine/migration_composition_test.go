package engine

import (
	"bytes"
	"context"
	"image/color"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

// Preserve composition.rs oracles with inline styles under ADR-013.
func TestMigrationEqualZIndexPreservesTreeOrder(t *testing.T) {
	img := rendered(t, 64, 64, `<div style="position:absolute;left:0;top:0;width:40px;height:40px;background:black;z-index:1"></div><div style="position:absolute;left:20px;top:0;width:40px;height:40px;background:white;z-index:1"></div>`)
	pixel(t, img, 10, 10, color.RGBA{A: 255})
	pixel(t, img, 30, 10, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

func TestMigrationOpaqueOverlapRepeatsEveryRGBAPixel(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	const source = `<div style="position:absolute;left:0;top:0;width:40px;height:40px;background:black;z-index:2"></div><div style="position:absolute;left:20px;top:20px;width:40px;height:40px;background:white;z-index:1"></div>`
	for _, size := range []display.Size{{Width: 64, Height: 64}, {Width: 800, Height: 480}} {
		first, err := r.RenderRGBA(context.Background(), size, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		pixel(t, first.Image, 30, 30, color.RGBA{A: 255})
		pixel(t, first.Image, 50, 50, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		second, err := r.RenderRGBA(context.Background(), size, []byte(source))
		if err != nil || !bytes.Equal(first.Image.Pix, second.Image.Pix) {
			t.Fatal("complete RGBA render changed on repetition", err)
		}
	}
}
