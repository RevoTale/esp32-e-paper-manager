package engine

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"testing"
)

func migrationText(t *testing.T, css, text string) *image.RGBA {
	t.Helper()
	return rendered(t, 256, 112, `<div style="font-family:sans-serif;font-size:20px;line-height:28px;color:black;`+css+`">`+text+`</div>`)
}

// Preserve text.rs's explicit-break and independent rectangular clip oracles.
func TestMigrationPreWrapPreservesNewlinesAndSoftWraps(t *testing.T) {
	wrapped := migrationText(t, "width:90px;white-space:pre-wrap", "MMMM MMMM\nAA")
	expected := migrationText(t, "width:90px", "MMMM<br>MMMM<br>AA")
	if !migrationInk(wrapped, image.Rect(0, 56, 256, 84)) {
		t.Fatal("missing third-line ink")
	}
	if !bytes.Equal(wrapped.Pix, expected.Pix) {
		t.Fatal("pre-wrap newline/soft wrap differs from explicit breaks")
	}
}

func TestMigrationHiddenOverflowClipsGlyphsAtBothEdges(t *testing.T) {
	const text = "MMMM MMMM<br>MMMM MMMM"
	visible := migrationText(t, "white-space:nowrap", text)
	for _, bound := range []image.Point{{X: 45, Y: 12}, {X: 90, Y: 28}, {X: 90, Y: 40}} {
		clipped := migrationText(t, fmt.Sprintf("width:%dpx;height:%dpx;white-space:nowrap;overflow:hidden", bound.X, bound.Y), text)
		expected := image.NewRGBA(visible.Bounds())
		copy(expected.Pix, visible.Pix)
		for y := range 112 {
			for x := range 256 {
				if x >= bound.X || y >= bound.Y {
					expected.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
				}
			}
		}
		if !migrationInk(expected, image.Rectangle{Max: bound}) || bytes.Equal(expected.Pix, visible.Pix) {
			t.Fatal("fixture did not retain and clip glyph pixels")
		}
		if !bytes.Equal(clipped.Pix, expected.Pix) {
			t.Fatalf("independent rectangular glyph clip differs at %v", bound)
		}
	}
}

func migrationInk(img *image.RGBA, area image.Rectangle) bool {
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			if img.RGBAAt(x, y).R < 255 {
				return true
			}
		}
	}
	return false
}
