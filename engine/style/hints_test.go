package style

import (
	"errors"
	"math"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func TestDimensionHintsCannotOverrideCSSOrMutateSource(t *testing.T) {
	b, err := Parse("width:20px", 7)
	if err != nil {
		t.Fatal(err)
	}
	with, err := b.WithDimensionHints(40, 30)
	if err != nil {
		t.Fatal(err)
	}
	w, _ := with.Get("width")
	h, _ := with.Get("height")
	if w.Length.Value != 20 || h.Length.Value != 30 || with.Count() != b.Count() {
		t.Fatalf("wrong hints/cascade: %+v %+v", w, h)
	}
	if _, ok := b.Get("height"); ok {
		t.Fatal("mutated source block")
	}
	var diagnostic *renderdiag.Error
	if !errors.As(with.Error("height", renderdiag.InvalidValue), &diagnostic) || diagnostic.Source != renderdiag.HTML || diagnostic.Element != 7 {
		t.Fatal("fabricated CSS range for HTML hint")
	}

}

func TestDimensionHintsRejectInvalidNumbers(t *testing.T) {
	for _, invalid := range []float64{-1, 9000, math.NaN(), math.Inf(1)} {
		if _, err := (Block{}).WithDimensionHints(invalid, 0); err == nil {
			t.Fatal("unbounded hint accepted")
		}
	}
}
