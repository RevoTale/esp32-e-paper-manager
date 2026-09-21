package engine

import (
	"image/color"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
)

func TestAbsoluteAutoSizeStretchesBetweenInsets(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 200, Height: 100}, `<div id="a" style="position:absolute;left:10px;right:20px;top:5px;bottom:15px;padding:3px;border:1px solid"></div>`)
	expectBox(t, s, "a", geometry.Rect{X: 10, Y: 5, Width: 170, Height: 80})
}

func TestInlineBlockPaintsOwnBackgroundBeforeChild(t *testing.T) {
	img := rendered(t, 80, 60, `<span style="display:inline-block;width:80px;height:60px;background:red"><div style="height:30px;background:blue"></div></span>`)
	pixel(t, img, 10, 10, color.RGBA{B: 255, A: 255})
	pixel(t, img, 10, 50, color.RGBA{R: 255, A: 255})
}

func TestStaticOpacityIgnoresZIndex(t *testing.T) {
	img := rendered(t, 80, 60, `<div style="position:relative;height:60px"><div style="position:absolute;z-index:1;width:80px;height:60px;background:blue"></div><div style="opacity:0.5;z-index:999;height:60px;background:red"></div></div>`)
	pixel(t, img, 10, 10, color.RGBA{B: 255, A: 255})
}

func TestInlineBlockDoesNotTrapPositionedDescendantZIndex(t *testing.T) {
	img := rendered(t, 80, 60, `<span style="display:inline-block;width:80px;height:60px;background:red"><div style="position:absolute;z-index:3;top:0;left:0;width:80px;height:60px;background:lime"></div></span><div style="position:absolute;z-index:2;top:0;left:0;width:80px;height:60px;background:blue"></div>`)
	pixel(t, img, 10, 10, color.RGBA{G: 255, A: 255})
}

func TestRootPercentageHeightUsesDefiniteViewport(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 100, Height: 80}, `<html style="height:100%"><body id="body" style="height:100%;background:red"></body></html>`)
	expectBox(t, s, "body", geometry.Rect{Width: 100, Height: 80})
}

func TestInlineBlockIntrinsicIncludesChildDimensionsAndEdges(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 200, Height: 100}, `<div id="first" style="display:inline-block"><div style="width:60px;height:20px;padding:3px;border:1px solid"></div></div><div id="next" style="display:inline-block;width:20px;height:20px"></div>`)
	first, next := s.elements[s.document.ByID["first"]], s.elements[s.document.ByID["next"]]
	if first.box.ContentWidth != 68 || next.x != 68 {
		t.Fatalf("first width %g, next x %g", first.box.ContentWidth, next.x)
	}
}

func TestAbsoluteSpanUsesBlockBox(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 100, Height: 80}, `<span id="a" style="position:absolute;left:10px;top:5px;width:30px;height:20px;background:red"></span>`)
	e := s.elements[s.document.ByID["a"]]
	if e.css.Keyword("display") != "block" {
		t.Fatalf("absolute span display %q", e.css.Keyword("display"))
	}
	img := rendered(t, 100, 80, `<span style="position:absolute;left:10px;top:5px;width:30px;height:20px;background:red"></span>`)
	pixel(t, img, 15, 10, color.RGBA{R: 255, A: 255})
}

func TestRelativeInlinePercentageUsesBlockContainingBox(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 200, Height: 100}, `<div style="width:100px"><span><b id="a" style="position:relative;left:50%">A</b></span></div>`)
	e := s.elements[s.document.ByID["a"]]
	if len(e.fragments) != 1 || e.fragments[0].X != 50 {
		t.Fatalf("relative fragment %+v", e.fragments)
	}
}
