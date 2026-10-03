package engine

import (
	"context"
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func TestReservedCornerIsRectangleNotBottomStrip(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.RenderRGBAWithOptions(context.Background(), display.Size{Width: 80, Height: 40}, []byte(`<div style="width:100vw;height:100vh;background:black"></div>`), Options{Reserved: image.Rect(60, 30, 80, 40)})
	if err != nil {
		t.Fatal(err)
	}
	pixel(t, result.Image, 59, 39, color.RGBA{A: 255})
	pixel(t, result.Image, 79, 29, color.RGBA{A: 255})
	pixel(t, result.Image, 60, 30, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	if len(result.Warnings) != 1 || result.Warnings[0].Code != renderdiag.ReservedOverlap {
		t.Fatalf("warnings %+v", result.Warnings)
	}
}

func TestReservedRegionRejectsOutsideViewport(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, area := range []image.Rectangle{image.Rect(-1, 0, 10, 10), image.Rect(0, 0, 81, 10), {Min: image.Pt(10, 10), Max: image.Pt(5, 5)}} {
		_, err := r.RenderRGBAWithOptions(context.Background(), display.Size{Width: 80, Height: 40}, nil, Options{Reserved: area})
		if !errors.Is(err, display.ErrFrameGeometry) {
			t.Fatalf("%v: %v", area, err)
		}
	}
}
