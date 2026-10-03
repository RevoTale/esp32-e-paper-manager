package engine

import (
	"image/color"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func TestInlineFragmentsMergeOncePerLine(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 200, Height: 80}, `<span id="s" style="background:rgba(255,0,0,0.5)">one <b>two</b> three</span>`)
	e := s.elements[s.document.ByID["s"]]
	if len(e.fragments) != 1 {
		t.Fatalf("one line produced %d backgrounds", len(e.fragments))
	}
}

func TestInlineEdgesAdvanceTextAndPaintVerticalPadding(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 200, Height: 80}, `<div style="padding-top:10px"><span id="s" style="padding:3px 5px;border:1px solid red;margin-left:2px">A</span><b id="b">B</b></div>`)
	side, next := s.elements[s.document.ByID["s"]], s.elements[s.document.ByID["b"]]
	if side.box.Padding.Left != 5 || side.box.Border.Left != 1 || side.fragments[0].X != 2 {
		t.Fatalf("inline box %+v fragments %+v", side.box, side.fragments)
	}
	if next.fragments[0].X < 23 {
		t.Fatalf("inline padding did not advance following text: %+v", next.fragments)
	}
	img := rendered(t, 200, 80, `<div style="padding-top:10px"><span style="padding:3px 5px;border:1px solid red;margin-left:2px">A</span></div>`)
	pixel(t, img, 3, 6, color.RGBA{R: 255, A: 255})
}

func TestPositionedInlineContainingBoxUsesFragment(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 200, Height: 80}, `<div style="padding:20px"><span id="s" style="position:relative">abc<span id="a" style="position:absolute;left:0;top:0;width:5px;height:5px"></span></span></div>`)
	side, absolute := s.elements[s.document.ByID["s"]], s.elements[s.document.ByID["a"]]
	if absolute.x != side.fragments[0].X || absolute.y != side.fragments[0].Y {
		t.Fatalf("absolute %g,%g inline %+v", absolute.x, absolute.y, side.fragments)
	}
}
