package dom

import (
	"errors"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func TestOneHTML5TreeOwnsEntitiesAndElementOrder(t *testing.T) {
	doc, err := Parse([]byte(`<div id="box" style="width:50%">Привіт &amp; <b>світ</b><img src="data:image/png;base64,AAAA" alt="image"></div>`))
	if err != nil {
		t.Fatal(err)
	}
	box := doc.ByID["box"]
	if box == nil || box.Ordinal != 4 || box.Children[0].Text != "Привіт & " || box.Parent.Tag != "body" {
		t.Fatalf("wrong canonical tree: %+v", box)
	}
	if doc.Root.Children[0].Tag != "html" || box.Children[1].Parent != box {
		t.Fatal("wrong tree ownership")
	}
	width, ok := box.Style.Get("width")
	if !ok || width.Length.Value != 50 {
		t.Fatal("inline style not attached to canonical node")
	}
}

func TestDocumentBoundaryRejectsUnsafeAndOversizedInput(t *testing.T) {
	inputs := []string{
		`<script>alert(1)</script>`, `<style>div{color:red}</style>`, `<svg/>`,
		`<img src="https://private/secret">`, `<div onclick="secret">`,
		`<div id="x" id="y">`, `<div id="x"></div><p id="x">`,
		`<body><body onload="secret">`, `<meta http-equiv="refresh" content="x">`,
		`<div style="width:NaNpx">`, `<div xmlns="x">`,
		strings.Repeat("<div>", 40), strings.Repeat("<br>", 1100),
		strings.Repeat(" ", 32769), string([]byte{0xff}),
	}
	for _, input := range inputs {
		_, err := Parse([]byte(input))
		if !errors.Is(err, renderdiag.ErrRejected) {
			t.Errorf("accepted hostile input %q: %v", input, err)
		}
		if err != nil && strings.Contains(err.Error(), "secret") {
			t.Fatal("source leaked")
		}
	}
}

func TestHTML5OptionalEndTagsAndMetadata(t *testing.T) {
	doc, err := Parse([]byte(`<!doctype html><!--hidden--><meta charset="UTF-8"><title>not painted</title><ul><li>one<li>two</ul><div data-epaper-bitmap="true"></div>`))
	if err != nil || doc == nil {
		t.Fatalf("%+v %v", doc, err)
	}
}
