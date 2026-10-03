package engine

import (
	"bytes"
	"context"
	_ "embed"
	"image/color"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

//go:embed testdata/viewport-review.html
var viewportReview []byte

func TestNativeIntegrationAcrossViewportSizes(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []display.Size{{Width: 250, Height: 122}, {Width: 400, Height: 300}, {Width: 800, Height: 480}, {Width: 480, Height: 800}} {
		result, err := r.RenderRGBA(context.Background(), size, viewportReview)
		if err != nil || len(result.Warnings) != 0 {
			t.Fatalf("%+v: %v, warnings %+v", size, err, result.Warnings)
		}
		if result.Image.Bounds().Dx() != size.Width || result.Image.Bounds().Dy() != size.Height {
			t.Fatalf("viewport changed: %+v", result.Image.Bounds())
		}
		pixel(t, result.Image, size.Width-20, 12, color.RGBA{A: 255})
		again, err := r.RenderRGBA(context.Background(), size, viewportReview)
		if err != nil || !bytes.Equal(result.Image.Pix, again.Image.Pix) {
			t.Fatalf("%+v: repeat render differs: %v", size, err)
		}
		result.Image.Pix[0] = 0
		if again.Image.Pix[0] != 255 {
			t.Fatal("returned renders share pixel storage")
		}
	}
}
