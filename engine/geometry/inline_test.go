package geometry

import "testing"

func TestInlineResolvesOnlyDecorations(t *testing.T) {
	css := boxStyle(t, `width:100px;height:100px;padding:10%;margin:2px auto;border:1px solid`)
	box, err := ResolveInline(css, Reference{Width: 80, Height: 40})
	if err != nil || box.ContentWidth != 0 || box.ContentHeight != 0 || box.Padding.Left != 8 || box.Padding.Top != 8 || box.Margin.Left != 0 || box.Margin.Top != 2 || box.Border.Left != 1 {
		t.Fatalf("%+v %v", box, err)
	}
}

func TestInlineRejectsInvalidReferencesAndEdges(t *testing.T) {
	css := boxStyle(t, `padding:10%`)
	for _, ref := range []Reference{{Width: -1}, {Height: -1}, {Width: 1e100}} {
		if _, err := ResolveInline(css, ref); err == nil {
			t.Fatal("invalid inline reference")
		}
	}
	if _, err := ResolveInline(boxStyle(t, `padding:8192em`), Reference{Width: 80}); err == nil {
		t.Fatal("resolved inline edge overflow")
	}
}
