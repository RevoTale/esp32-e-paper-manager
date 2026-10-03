package engine

import (
	"bytes"
	"testing"
)

func TestDecoratedForcedLineBreakKeepsSecondLineGlyphs(t *testing.T) {
	const source = `<div style="padding:8px;font-size:12px;line-height:14px"><span style="border:1px solid transparent">Line one<br>Line two</span></div>`
	img := rendered(t, 180, 70, source)
	want := rendered(t, 180, 70, `<div style="padding:8px;font-size:12px;line-height:14px"><br>Line two</div>`)
	for y := 22; y < 35; y++ {
		if !bytes.Equal(img.Pix[y*img.Stride:(y+1)*img.Stride], want.Pix[y*want.Stride:(y+1)*want.Stride]) {
			t.Fatalf("second-line RGBA row %d differs from plain text control", y)
		}
	}
	ink := 0
	for y := 23; y < 34; y++ {
		for x := 12; x < 48; x++ {
			if img.RGBAAt(x, y).R < 128 {
				ink++
			}
		}
	}
	if ink < 20 {
		t.Fatalf("second line contains only %d dark text pixels", ink)
	}
}
