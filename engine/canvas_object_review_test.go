package engine

import (
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/tdewolff/canvas"
)

// This active reproduction pins dae8cd8e19a7's merge defect. Re-evaluate the
// adapter when changing Canvas: a corrected upstream must update this oracle.
func TestPinnedCanvasSharedFaceMergesTextIntoObjectSpan(t *testing.T) {
	mixed := false
	canvasBoundaryText(t, false).WalkLines(func(_ float64, spans []canvas.TextSpan) {
		for _, span := range spans {
			mixed = mixed || !span.IsText() && strings.Contains(span.Text, "Line two")
		}
	})
	if !mixed {
		t.Fatal("pinned Canvas merge behavior changed; re-evaluate the object-face adapter")
	}
}

func TestCanvasObjectFaceIdentityPreservesAllText(t *testing.T) {
	var text strings.Builder
	canvasBoundaryText(t, true).WalkLines(func(_ float64, spans []canvas.TextSpan) {
		for _, span := range spans {
			if span.IsText() {
				text.WriteString(span.Text)
			}
		}
	})
	if text.String() != "Line one\nLine two" {
		t.Fatalf("text escaped into object spans: %q", text.String())
	}
}

func canvasBoundaryText(t *testing.T, separate bool) *canvas.Text {
	t.Helper()
	s := layoutDocument(t, display.Size{Width: 180, Height: 70}, `<span id="s" style="font-size:12px">text</span>`)
	face := s.fonts.face(elementID(s, "s").css)
	rich := canvas.NewRichText(face)
	boundary := func() {
		if separate {
			copy := *face
			rich.SetFace(&copy)
		}
		rich.WriteCanvas(canvas.New(1, 0), canvas.Baseline)
		rich.SetFace(face)
	}
	boundary()
	rich.WriteString("Line one\nLine two")
	boundary()
	return rich.ToText(164, 0, canvas.Left, canvas.Top, &canvas.TextOptions{Linebreaker: canvas.GreedyLinebreaker{}})
}
