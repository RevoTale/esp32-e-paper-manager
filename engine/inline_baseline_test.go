package engine

import (
	"image/color"
	"math"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func elementID(s *scene, id string) *element { return s.elements[s.document.ByID[id]] }

func TestInlineBlockUsesLastInFlowTextBaseline(t *testing.T) {
	for _, contents := range []string{
		`<span id="last">X</span>`,
		`first<br><span id="last">X</span>`,
		`<div>first</div><div><span id="last">X</span></div><div style="height:7px"></div>`,
		`<ul><li>first<br><span id="last">X</span></li></ul>`,
	} {
		t.Run(contents, func(t *testing.T) {
			s := layoutDocument(t, display.Size{Width: 240, Height: 160}, `<div style="font-size:20px;line-height:26px"><span id="outer">X</span><span id="box" style="display:inline-block;padding:3px;border:2px solid;margin:4px">`+contents+`</span></div>`)
			outer, last := elementID(s, "outer"), elementID(s, "last")
			if math.Abs(outer.fragments[0].Y-last.fragments[0].Y) > 1e-9 {
				t.Fatalf("equal-font text baselines differ: outer %+v last %+v", outer.fragments, last.fragments)
			}
		})
	}
}

func TestInlineBlockBottomMarginBaselineFallback(t *testing.T) {
	for _, contents := range []string{
		``, `<div style="height:4px"></div>`,
		`<span style="position:absolute">out of flow</span>`,
	} {
		t.Run(contents, func(t *testing.T) { assertBottomBaseline(t, "visible", contents) })
	}
	for _, overflow := range []string{"hidden", "clip"} {
		t.Run(overflow, func(t *testing.T) { assertBottomBaseline(t, overflow, "text") })
	}
}

func assertBottomBaseline(t *testing.T, overflow, contents string) {
	t.Helper()
	s := layoutDocument(t, display.Size{Width: 240, Height: 160}, `<div style="font-size:20px;line-height:26px"><span id="outer">X</span><span id="box" style="display:inline-block;width:60px;height:50px;padding:3px;border:2px solid;margin:4px;overflow:`+overflow+`">`+contents+`</span></div>`)
	outer, box := elementID(s, "outer"), elementID(s, "box")
	baseline := outer.fragments[0].Y + s.fonts.face(outer.css).Metrics().Ascent
	bottom := box.borderRect().Bottom() + box.box.Margin.Bottom
	if math.Abs(baseline-bottom) > 1e-9 {
		t.Fatalf("baseline %g != bottom margin edge %g", baseline, bottom)
	}
}

func TestInlineBlockVisibleOverflowBaselineMayExceedHeight(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 240, Height: 160}, `<div style="font-size:20px;line-height:26px"><span id="outer">X</span><span style="display:inline-block;height:1px">first<br><span id="last">X</span></span></div>`)
	outer, last := elementID(s, "outer"), elementID(s, "last")
	if math.Abs(outer.fragments[0].Y-last.fragments[0].Y) > 1e-9 {
		t.Fatalf("overflow baseline mismatch: outer %+v last %+v", outer.fragments, last.fragments)
	}
}

func TestInlineBlockTextBaselineControlsSiblingPixels(t *testing.T) {
	img := rendered(t, 240, 100, `<div style="font-size:20px;line-height:26px;color:transparent"><span style="background:lime">X</span><span style="display:inline-block;padding:3px;border:2px solid red;margin:4px">X</span></div>`)
	pixel(t, img, 5, 15, color.RGBA{G: 255, A: 255})
	pixel(t, img, 5, 40, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}
