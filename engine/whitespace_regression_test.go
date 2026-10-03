package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func TestPreLineDropsSpaceAfterForcedBreakAcrossSpans(t *testing.T) {
	for _, between := range []string{"\n   ", "<br>   ", "\n<span>  </span>"} {
		s := layoutDocument(t, display.Size{Width: 200, Height: 100}, `<div style="white-space:pre-line">X`+between+`<span id="next"> Y</span></div>`)
		next := s.elements[s.document.ByID["next"]]
		if len(next.fragments) != 1 || next.fragments[0].X != 0 {
			t.Fatalf("%q: %+v", between, next.fragments)
		}
	}
}

func TestPreservedTabsRejectInsteadOfUsingWrongStops(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"pre", "pre-wrap"} {
		_, err = r.RenderRGBA(context.Background(), display.Size{Width: 200, Height: 100}, []byte(`<div style="white-space:`+mode+`">A`+"\t"+`B</div>`))
		var diagnostic *renderdiag.Error
		if !errors.As(err, &diagnostic) || diagnostic.Code != renderdiag.InvalidValue {
			t.Fatal(mode, err)
		}
	}
}
