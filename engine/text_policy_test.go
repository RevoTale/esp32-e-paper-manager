package engine

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func TestNowrapAlignmentStillUsesContainerWidth(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 200, Height: 100}, `<div style="white-space:nowrap;text-align:right"><span id="text">hello</span></div>`)
	e := s.elements[s.document.ByID["text"]]
	if len(e.fragments) != 1 || e.fragments[0].X <= 100 || math.Abs(e.fragments[0].Right()-200) > .01 {
		t.Fatalf("nowrap right alignment: %+v", e.fragments)
	}
}

func TestEllipsisPreservesHiddenRunLineMetrics(t *testing.T) {
	var heights []float64
	for _, overflow := range []string{"clip", "ellipsis"} {
		s := layoutDocument(t, display.Size{Width: 200, Height: 200}, `<div style="width:40px;font-size:16px;white-space:nowrap;overflow:hidden;text-overflow:`+overflow+`">abcdefghij<span style="font-size:128px">Z</span></div><div id="next" style="height:5px"></div>`)
		heights = append(heights, s.elements[s.document.ByID["next"]].y)
	}
	if heights[0] != heights[1] {
		t.Fatalf("ellipsis changed subsequent layout: %v", heights)
	}
}

func TestMixedInlineWhitespaceRejectsInsteadOfSilentlyWrapping(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.RenderRGBA(context.Background(), display.Size{Width: 100, Height: 80}, []byte(`<div style="width:35px"><span style="white-space:nowrap">one two</span></div>`))
	var diagnostic *renderdiag.Error
	if !errors.As(err, &diagnostic) || diagnostic.Code != renderdiag.InvalidValue || diagnostic.Source != renderdiag.Inline || diagnostic.End == 0 {
		t.Fatalf("missing unsupported inline-mode diagnostic: %v", err)
	}
	// The same declaration is supported on its own block formatting context.
	img := rendered(t, 100, 80, `<span style="display:inline-block;width:35px;white-space:nowrap">one two</span>`)
	if img == nil {
		t.Fatal("missing inline-block result")
	}
}

func TestEmergencyWrappingKeepsCombiningGraphemesTogether(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 35, Height: 400}, `<div id="text" style="font-size:20px;line-height:24px;overflow-wrap:anywhere">áááááááááá</div>`)
	e := s.elements[s.document.ByID["text"]]
	if e.box.ContentHeight < 72 {
		t.Fatalf("unbroken long word height %g", e.box.ContentHeight)
	}
	for _, paint := range e.glyphs {
		if paint.path.FastBounds().W() > 36 {
			t.Fatal("emergency line exceeds its viewport")
		}
	}
}

func TestEllipsisProducesBoundedSingleLine(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 80, Height: 100}, `<div id="text" style="font-size:20px;line-height:24px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">abcdef <b>ghijklmnop</b></div>`)
	e := s.elements[s.document.ByID["text"]]
	if e.box.ContentHeight != 24 {
		t.Fatalf("ellipsis changed line count: %g", e.box.ContentHeight)
	}
	if len(s.warnings) == 0 {
		t.Fatal("ellipsis did not report removed content")
	}
}

func TestEllipsisDoesNotDropInlineBoxEdges(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.RenderRGBA(context.Background(), display.Size{Width: 80, Height: 60}, []byte(`<div style="width:40px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis"><span style="padding:0 8px;border:1px solid black">abcdefghij</span></div>`))
	var diagnostic *renderdiag.Error
	if !errors.As(err, &diagnostic) || diagnostic.Code != renderdiag.InvalidValue {
		t.Fatal("elision silently discarded inline edges", err)
	}
}
