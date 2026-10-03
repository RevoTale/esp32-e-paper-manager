package style

import (
	"image/color"
	"testing"
)

func computedFixture(t *testing.T, tag, source string, parent *Computed) Computed {
	t.Helper()
	block, err := Parse(source, 1)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Compute(tag, block, parent, Metrics{ViewportWidth: 800, ViewportHeight: 480})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestComputedInheritanceAndFontRelativeBases(t *testing.T) {
	root := computedFixture(t, "html", "font-size:2rem;color:red;background:black;margin:4px;line-height:1.5", nil)
	child := computedFixture(t, "div", "font-size:2em", &root)
	grandchild := computedFixture(t, "span", "font-size:50%", &child)
	if root.FontSize() != 32 || child.FontSize() != 64 || grandchild.FontSize() != 32 || child.RootFont() != 32 {
		t.Fatal("font-size used a circular/local or incorrect root basis")
	}
	if child.Color("color") != (color.NRGBA{255, 0, 0, 255}) || child.Color("background-color").A != 0 || child.Length("margin-top").Value != 0 {
		t.Fatal("non-inherited properties leaked")
	}
	if child.Number("line-height") != 1.5 {
		t.Fatal("unitless line height was prematurely made absolute")
	}
}

func TestInheritedLengthLineHeightIsComputedOnce(t *testing.T) {
	parent := computedFixture(t, "html", "font-size:20px;line-height:2em", nil)
	child := computedFixture(t, "p", "font-size:10px", &parent)
	if child.Length("line-height") != (Length{Value: 40}) {
		t.Fatalf("wrong inherited line-height: %+v", child.Length("line-height"))
	}
}

func TestDefaultsAndCurrentColor(t *testing.T) {
	root := computedFixture(t, "html", "color:blue", nil)
	child := computedFixture(t, "strong", "color:currentColor;border:1px solid", &root)
	if child.Keyword("display") != "inline" || child.Keyword("font-weight") != "bold" || child.Color("border-left-color") != root.Color("color") {
		t.Fatal("semantic defaults/currentColor broken")
	}
	if computedFixture(t, "head", "", &root).Keyword("display") != "none" {
		t.Fatal("head paints")
	}
	if computedFixture(t, "pre", "", &root).Keyword("white-space") != "pre" {
		t.Fatal("pre does not preserve whitespace")
	}
}

func TestComputedFontLimitsRejectRatherThanShrink(t *testing.T) {
	for _, source := range []string{"font-size:7px", "font-size:129px", "font-size:8192em", "line-height:8192em"} {
		block, err := Parse(source, 1)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compute("html", block, nil, Metrics{}); err == nil {
			t.Errorf("accepted %s", source)
		}
	}
}
