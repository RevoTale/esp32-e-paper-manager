package engine

import (
	"image"
	"image/color"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/tdewolff/canvas"
	"golang.org/x/image/font/gofont/goregular"
)

func TestSurfaceUsesExactLogicalViewport(t *testing.T) {
	for _, size := range []display.Size{{Width: 800, Height: 480}, {Width: 250, Height: 122}, {Width: 17, Height: 9}} {
		s, err := newSurface(size)
		if err != nil {
			t.Fatal(err)
		}
		s.context.SetFillColor(color.Black)
		s.context.DrawPath(0, 0, canvas.Rectangle(4, 3))
		if s.image.Bounds() != image.Rect(0, 0, size.Width, size.Height) {
			t.Fatal("viewport was scaled")
		}
		if s.image.RGBAAt(1, 1) != (color.RGBA{A: 255}) || s.image.RGBAAt(1, size.Height-1) != (color.RGBA{255, 255, 255, 255}) {
			t.Fatal("top-left coordinate or background changed")
		}
	}
}

func TestSurfaceRejectsUnsafeDimensions(t *testing.T) {
	for _, size := range []display.Size{{}, {Width: -1, Height: 5}, {Width: 2049, Height: 1}, {Width: 2048, Height: 2048}, {Width: 1, Height: int(^uint(0) >> 1)}} {
		if _, err := newSurface(size); err == nil {
			t.Fatal("unsafe allocation accepted", size)
		}
	}
}

func TestSurfaceNativeGoUkrainianText(t *testing.T) {
	s, err := newSurface(display.Size{Width: 320, Height: 80})
	if err != nil {
		t.Fatal(err)
	}
	family := canvas.NewFontFamily("Go")
	if err := family.LoadFont(goregular.TTF, 0, canvas.FontRegular); err != nil {
		t.Fatal(err)
	}
	face := family.Face(20*72/25.4, color.Black)
	s.context.DrawText(4, 30, canvas.NewTextLine(face, "Привіт, Україно!", canvas.Left))
	ink := 0
	for y := 0; y < 60; y++ {
		for x := 0; x < 300; x++ {
			if s.image.RGBAAt(x, y).R < 128 {
				ink++
			}
		}
	}
	if ink < 100 {
		t.Fatal("text did not paint", ink)
	}
}
