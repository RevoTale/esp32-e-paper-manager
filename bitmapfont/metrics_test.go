package bitmapfont

import "testing"

func TestMetricsAndWrapping(t *testing.T) {
	if got := ForSize(8); got.Advance != 5 || got.Height != 8 || got.LineHeight != 10 {
		t.Fatalf("ForSize(8) = %#v", got)
	}
	if got := ForSize(20); got.Width != 10 || got.Advance != 11 {
		t.Fatalf("ForSize(20) = %#v", got)
	}
	if !glyphPainted('A', ForSize(13)) {
		t.Fatal("RunePixel(A) produced an empty glyph")
	}
	if got := WrappedLines([]byte("123456"), 15, 8); got != 2 {
		t.Fatalf("WrappedLines() = %d", got)
	}
	if width, height := TimestampDimensions(); width != 120 || height != 21 {
		t.Fatalf("TimestampDimensions() = %dx%d", width, height)
	}
}

func TestGlyphDecoding(t *testing.T) {
	if got := GlyphCount([]byte("A&amp;Б")); got != 3 {
		t.Fatalf("GlyphCount() = %d", got)
	}
	if glyph, next := NextGlyph([]byte(" \nA"), 0); glyph != ' ' || next != 2 {
		t.Fatalf("NextGlyph() = %q, %d", glyph, next)
	}
	if glyph, next := NextGlyph([]byte("&broken"), 0); glyph != '\ufffd' || next != 7 {
		t.Fatalf("NextGlyph(malformed entity) = %q, %d", glyph, next)
	}
	if glyph, next := NextGlyph([]byte("x"), 1); glyph != '\ufffd' || next != 1 {
		t.Fatalf("NextGlyph(end) = %q, %d", glyph, next)
	}
	if glyph, next := NextGlyph([]byte("\t\rX"), 0); glyph != ' ' || next != 2 {
		t.Fatalf("NextGlyph(whitespace) = %q, %d", glyph, next)
	}
}

func TestRunePixelBoundsAndReplacement(t *testing.T) {
	metrics := ForSize(13)
	if RunePixel('A', -1, 0, metrics) || RunePixel('A', metrics.Width, 0, metrics) {
		t.Fatal("RunePixel painted outside glyph bounds")
	}
	if !glyphPainted('Б', metrics) {
		t.Fatal("replacement glyph is empty")
	}
}

func glyphPainted(value rune, metrics Metrics) bool {
	for y := 0; y < metrics.Height; y++ {
		for x := 0; x < metrics.Width; x++ {
			if RunePixel(value, x, y, metrics) {
				return true
			}
		}
	}
	return false
}
