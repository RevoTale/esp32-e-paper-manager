package engine

import (
	"context"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine/dom"
	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
)

func layoutDocument(t *testing.T, size display.Size, source string) *scene {
	t.Helper()
	doc, err := dom.Parse([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	a, err := loadAssets(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	f, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	s, err := layoutScene(context.Background(), size, doc, a, f)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func expectBox(t *testing.T, s *scene, id string, want geometry.Rect) {
	t.Helper()
	got := s.elements[s.document.ByID[id]].borderRect()
	if got != want {
		t.Fatalf("%s: box %+v, want %+v", id, got, want)
	}
}

func TestBlockLayoutAdaptsBeforePainting(t *testing.T) {
	for _, width := range []int{800, 250, 17} {
		s := layoutDocument(t, display.Size{Width: width, Height: 480}, `<div id="outer" style="width:100%;box-sizing:border-box;padding:2px"><div id="a" style="width:50%;height:20px;margin-bottom:3px"></div><div id="b" style="height:10px;margin-top:4px"></div></div>`)
		expectBox(t, s, "outer", geometry.Rect{Width: float64(width), Height: 41})
		expectBox(t, s, "a", geometry.Rect{X: 2, Y: 2, Width: float64(width-4) / 2, Height: 20})
		expectBox(t, s, "b", geometry.Rect{X: 2, Y: 29, Width: float64(width - 4), Height: 10})
	}
}

func TestRelativeMovementDoesNotChangeSiblingFlow(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 200, Height: 100}, `<div id="a" style="position:relative;left:10px;top:7px;height:20px"><div id="child" style="height:5px"></div></div><div id="b" style="height:9px"></div>`)
	expectBox(t, s, "a", geometry.Rect{X: 10, Y: 7, Width: 200, Height: 20})
	expectBox(t, s, "child", geometry.Rect{X: 10, Y: 7, Width: 200, Height: 5})
	expectBox(t, s, "b", geometry.Rect{Y: 20, Width: 200, Height: 9})
}

func TestAbsoluteUsesPositionedPaddingBoxAndStaysOutOfFlow(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 300, Height: 200}, `<div id="host" style="position:relative;margin:10px;width:100px;height:80px;padding:5px;border:2px solid"><div><div id="absolute" style="position:absolute;right:0;bottom:0;width:20px;height:10px"></div></div></div><div id="following" style="height:3px"></div>`)
	expectBox(t, s, "absolute", geometry.Rect{X: 102, Y: 92, Width: 20, Height: 10})
	expectBox(t, s, "following", geometry.Rect{Y: 114, Width: 300, Height: 3})
}

func TestHiddenMetadataNeverPaintsOrConsumesFlow(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 100, Height: 100}, `<head style="display:block;height:80px"><title>hidden</title></head><body><div id="a" style="display:none;height:40px"></div><div id="b" style="height:5px"></div></body>`)
	expectBox(t, s, "b", geometry.Rect{Width: 100, Height: 5})
}
