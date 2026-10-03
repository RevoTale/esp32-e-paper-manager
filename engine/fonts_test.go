package engine

import (
	"math"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

func TestEmbeddedFacesUseExactCSSPixelsAndVariants(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		"font-size:24px", "font-size:24px;font-weight:bold", "font-size:24px;font-style:italic",
		"font-size:24px;font-weight:bold;font-style:italic", "font-family:monospace;font-size:24px",
		"font-family:monospace;font-weight:bold;font-size:24px", "font-family:monospace;font-style:italic;font-size:24px",
		"font-family:monospace;font-weight:bold;font-style:italic;font-size:24px",
	} {
		block, err := style.Parse(source, 1)
		if err != nil {
			t.Fatal(err)
		}
		computed, err := style.Compute("span", block, nil, style.Metrics{})
		if err != nil {
			t.Fatal(err)
		}
		face := fonts.face(computed)
		// Canvas's reciprocal pt/mm conversions incur floating-point rounding;
		// tolerate 1e-9 px, not a real scale or DPI discrepancy.
		if math.Abs(face.Size-24) > 1e-9 || face.TextWidth("Привіт") <= 20 || face.Metrics().Ascent <= 0 {
			t.Fatalf("bad metrics: %s %+v", source, face.Metrics())
		}
		for _, glyph := range face.Glyphs("Привіт") {
			if glyph.ID == 0 {
				t.Fatal("embedded font lacks expected Cyrillic glyph")
			}
		}
	}
}
