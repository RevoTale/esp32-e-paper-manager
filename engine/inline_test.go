package engine

import (
	"math"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func TestMixedInlineTextWrapsWithFontMetrics(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 160, Height: 200}, `<div id="line" style="font-size:20px;line-height:24px">one <b id="bold">two</b> three four five six</div><div id="after" style="height:5px"></div>`)
	line := s.elements[s.document.ByID["line"]]
	bold := s.elements[s.document.ByID["bold"]]
	if line.box.ContentHeight < 48 || math.Mod(line.box.ContentHeight, 24) != 0 {
		t.Fatalf("line height %g", line.box.ContentHeight)
	}
	if len(bold.glyphs) == 0 || len(bold.fragments) == 0 || bold.css.Keyword("font-weight") != "bold" {
		t.Fatal("mixed face lost its owner, paint or geometry")
	}
	after := s.elements[s.document.ByID["after"]]
	if after.y != line.box.ContentHeight {
		t.Fatalf("following block at %g, text ends %g", after.y, line.box.ContentHeight)
	}
}

func TestWhitespaceAndExplicitBreaks(t *testing.T) {
	for _, tc := range []struct {
		html   string
		height float64
	}{
		{`a   <span> b </span> c`, 20},
		{`a<br>b`, 40},
		{"<pre>a\n b</pre>", 40},
	} {
		s := layoutDocument(t, display.Size{Width: 250, Height: 100}, `<div id="line" style="line-height:20px">`+tc.html+`</div>`)
		if got := s.elements[s.document.ByID["line"]].box.ContentHeight; got != tc.height {
			t.Errorf("%q: height %g, want %g", tc.html, got, tc.height)
		}
	}
}

func TestInlineBlocksAreAtomicAndWrap(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 100, Height: 160}, `<div id="line"><span id="a" style="display:inline-block;width:60px;height:20px"></span> <span id="b" style="display:inline-block;width:60px;height:30px"></span></div>`)
	a, b := s.elements[s.document.ByID["a"]], s.elements[s.document.ByID["b"]]
	if a.x != 0 || b.x != 0 || b.y <= a.y || b.box.ContentWidth != 60 {
		t.Fatalf("inline blocks %+v, %+v", a.borderRect(), b.borderRect())
	}
}

func TestUnicodeTextIsShapedAndNeverByteSplit(t *testing.T) {
	s := layoutDocument(t, display.Size{Width: 160, Height: 200}, `<p id="text" style="font-size:24px">Привіт світе!</p>`)
	e := s.elements[s.document.ByID["text"]]
	if len(e.glyphs) == 0 || e.box.ContentHeight <= 0 {
		t.Fatal("Cyrillic has no shaped geometry")
	}
}
