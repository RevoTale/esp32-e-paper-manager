package dom

import (
	"fmt"
	"strings"
	"testing"
)

func TestFiveRelatedEdits(t *testing.T) {
	var source strings.Builder
	var edits []Edit
	for i := range 5 {
		id := fmt.Sprintf("item%d", i)
		fmt.Fprintf(&source, `<p id="%s">old</p>`, id)
		edits = append(edits, Edit{ID: id, Text: pointer("new " + id)})
	}
	result, err := Apply([]byte(source.String()), edits)
	if err != nil || strings.Count(string(result), "new item") != 5 || strings.Contains(string(result), "old") {
		t.Fatal(string(result), err)
	}
}

func pointer(s string) *string { return &s }

func TestAtomicEdits(t *testing.T) {
	source := []byte(`<div id="a"><span id="child">old</span></div><p id="b">keep</p>`)
	result, err := Apply(source, []Edit{
		{ID: "a", HTML: pointer(`<strong>Ї &amp; test</strong>`)},
		{ID: "b", Text: pointer(`<literal>`), Attributes: map[string]*string{"style": pointer("color:red")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), `<strong>Ї &amp; test</strong>`) || !strings.Contains(string(result), `&lt;literal&gt;`) {
		t.Fatal(string(result))
	}
	doc, err := Parse(result)
	if err != nil || doc.ByID["child"] != nil || doc.ByID["b"].Attrs["style"] != "color:red" {
		t.Fatal(doc, err)
	}
	removed, err := Apply(result, []Edit{{ID: "a", Remove: true}, {ID: "b", Attributes: map[string]*string{"style": nil}}})
	if err != nil || strings.Contains(string(removed), "strong") || strings.Contains(string(removed), "style=") {
		t.Fatal(string(removed), err)
	}
}

func TestRejectEditConflicts(t *testing.T) {
	source := []byte(`<div id="a"><span id="child">old</span></div><p id="b">keep</p>`)
	cases := [][]Edit{
		nil, {{ID: "missing", Remove: true}}, {{ID: "a"}},
		{{ID: "a", Remove: true}, {ID: "a", Text: pointer("x")}},
		{{ID: "a", Text: pointer("x")}, {ID: "child", Remove: true}},
		{{ID: "child", Remove: true}, {ID: "a", Text: pointer("x")}},
		{{ID: "a", Text: pointer("x"), HTML: pointer("x")}},
		{{ID: "a", Text: pointer("x"), Remove: true}},
		{{ID: "a", Attributes: map[string]*string{"id": pointer("changed")}}},
		{{ID: "a", Attributes: map[string]*string{"ID": pointer("changed")}}},
		{{ID: "a", Attributes: map[string]*string{`title="x" style`: pointer("display:none")}}},
		{{ID: "a", HTML: pointer(`<script>alert(1)</script>`)}},
		{{ID: "a", HTML: pointer(`<i id="b">duplicate</i>`)}},
		{{ID: "b", HTML: pointer(`<div>reparented</div>`)}},
	}
	for i, edits := range cases {
		if result, err := Apply(source, edits); err == nil || result != nil {
			t.Errorf("accepted %d: %s", i, result)
		}
	}
	if string(source) != `<div id="a"><span id="child">old</span></div><p id="b">keep</p>` {
		t.Fatal("input mutated")
	}
}
