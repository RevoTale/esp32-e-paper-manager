package refreshstamp

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"testing"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func TestLabelPixelsMatchIndependentFontDrawer(t *testing.T) {
	frame := testFrame(t, 128, 32, 0)
	label := testLabel(t)
	if err := Paint(frame, label); err != nil {
		t.Fatal(err)
	}
	want := image.NewGray(image.Rect(0, 0, 128, 32))
	draw.Draw(want, want.Bounds(), image.White, image.Point{}, draw.Src)
	for _, border := range []image.Rectangle{image.Rect(8, 11, 128, 12), image.Rect(8, 31, 128, 32), image.Rect(8, 11, 9, 32), image.Rect(127, 11, 128, 32)} {
		draw.Draw(want, border, image.Black, image.Point{}, draw.Src)
	}
	text := font.Drawer{Dst: want, Src: image.Black, Face: basicfont.Face7x13, Dot: fixed.P(12, 15+basicfont.Face7x13.Ascent)}
	text.DrawString("2026-09-06 15:34")
	for y := 0; y < 32; y++ {
		for x := 0; x < 128; x++ {
			black := want.GrayAt(x, y) == (color.Gray{Y: 0})
			if (frame.Pixel(x, y) == display.Black) != black {
				t.Fatalf("glyph/border mismatch at %d,%d", x, y)
			}
		}
	}
}

func TestPartialReusesExactConfirmedPixelsWithoutAllocatingPaint(t *testing.T) {
	tracker := testTracker(t)
	start := time.Date(2026, 9, 6, 12, 34, 0, 0, time.UTC)
	label, err := tracker.Begin(1, start)
	if err != nil {
		t.Fatal(err)
	}
	full := testFrame(t, 128, 32, 0)
	if err := Paint(full, label); err != nil {
		t.Fatal(err)
	}
	if err := tracker.Complete(1, start.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	partial := testFrame(t, 128, 32, 0)
	retained, err := tracker.ForPartial()
	if err != nil {
		t.Fatal(err)
	}
	allocations := testing.AllocsPerRun(50, func() {
		clear(partial.Bytes())
		if err := Paint(partial, retained); err != nil {
			panic(err)
		}
	})
	if allocations != 0 || !bytes.Equal(full.Bytes(), partial.Bytes()) {
		t.Fatalf("partial differs or painting allocates: %g", allocations)
	}
}
