package engine

import (
	"context"
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func TestDarkSolidFillHasNoPeriodicWhiteHoles(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, shade := range []string{"#111", "#333", "#555"} {
		result, err := r.RenderRGBA(context.Background(), display.Size{Width: 16, Height: 16},
			[]byte(`<div style="width:16px;height:16px;background:`+shade+`"></div>`))
		if err != nil {
			t.Fatal(err)
		}
		frame, err := result.Frame(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for y := 1; y < 15; y++ {
			for x := 1; x < 15; x++ {
				if frame.Pixel(x, y) != display.Black {
					t.Fatalf("%s: periodic white hole at %d,%d; RGBA=%v", shade, x, y, result.Image.RGBAAt(x, y))
				}
			}
		}
	}
}

func TestMonochromeModeKeepsDitherExplicit(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	result := Result{Image: img}
	plain, err := result.Frame(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range plain.Bytes() {
		if b != 0 {
			t.Fatalf("default gray fill contains halftone: %x", plain.Bytes())
		}
	}
	if _, err := result.FrameWithMode(context.Background(), MonochromeMode(255)); !errors.Is(err, display.ErrColor) {
		t.Fatalf("invalid mode: %v", err)
	}
}

func TestDarkTextOpaqueInkHasNoGridHoles(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.RenderRGBA(context.Background(), display.Size{Width: 300, Height: 70},
		[]byte(`<div style="font-size:36px;color:#111">Grain test 111</div>`))
	if err != nil {
		t.Fatal(err)
	}
	frame, err := result.Frame(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ink := 0
	for y := 0; y < 70; y++ {
		for x := 0; x < 300; x++ {
			if result.Image.RGBAAt(x, y) == (color.RGBA{R: 17, G: 17, B: 17, A: 255}) {
				ink++
				if frame.Pixel(x, y) != display.Black {
					t.Fatalf("opaque text ink lost at %d,%d", x, y)
				}
			}
		}
	}
	if ink < 100 {
		t.Fatalf("insufficient real text ink samples: %d", ink)
	}
}
