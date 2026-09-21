package dom

import (
	"strings"
	"testing"
)

func TestEditBoundsAndStructure(t *testing.T) {
	source := []byte(`<body id="body"><p id="a">old</p><br id="b"></body>`)
	cases := [][]Edit{
		{{ID: "body", Remove: true}},
		{{ID: "b", Text: pointer("x")}},
		{{ID: "a", HTML: pointer(`<body>drop</body>`)}},
		{{ID: "a", HTML: pointer(`<!doctype html>x`)}},
		{{ID: "a", Attributes: map[string]*string{"style lang": pointer("x")}}},
		{{ID: "a", Text: pointer("\xff")}},
		{{ID: "a", Attributes: map[string]*string{"title": pointer("\xff")}}},
		{{ID: "a", Text: pointer(strings.Repeat("&", 7000))}},
		{{ID: "a", Text: pointer(strings.Repeat("x", 32768))}},
	}
	for i, edits := range cases {
		if result, err := Apply(source, edits); err == nil || result != nil {
			t.Errorf("accepted %d: %s", i, result)
		}
	}
	if _, err := Apply([]byte(`<script>x</script>`), []Edit{{ID: "a", Remove: true}}); err == nil {
		t.Fatal("invalid original")
	}
	if _, err := Apply(source, make([]Edit, 65)); err == nil {
		t.Fatal("edit limit")
	}
	if result, err := Apply(source, []Edit{{ID: "a", HTML: pointer(`<!--ignored--><em>ok</em>`)}}); err != nil || strings.Contains(string(result), "ignored") {
		t.Fatal(string(result), err)
	}
}

func TestBoundedHTMLWriter(t *testing.T) {
	w := &boundedHTML{}
	if _, err := w.WriteString(strings.Repeat("x", 32768)); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteByte('x'); err == nil || w.Len() != 32768 {
		t.Fatal(err, w.Len())
	}
	if _, err := w.Write([]byte("overflow")); err == nil {
		t.Fatal("overflow")
	}
}

func TestRootAttributeEdit(t *testing.T) {
	result, err := Apply([]byte(`<body id="root"><p>keep</p></body>`), []Edit{{ID: "root", Attributes: map[string]*string{"style": pointer("background:red")}}})
	if err != nil || !strings.Contains(string(result), `<body id="root" style="background:red"><p>keep</p></body>`) {
		t.Fatal(string(result), err)
	}
}
