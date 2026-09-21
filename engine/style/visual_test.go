package style

import (
	"image/color"
	"testing"
)

func TestVisualAndTextPropertyGrammar(t *testing.T) {
	valid := []string{
		"display:inline-block", "position:absolute;left:10%;bottom:2rem", "box-sizing:border-box",
		"color:rebeccapurple;background-color:rgba(255,0,0,0.5)", "color:#1234", "color:currentColor",
		"z-index:-12;opacity:0.5;overflow:hidden", "font-family:Go, sans-serif;font-size:2em;font-weight:700;font-style:italic",
		"font-family:monospace;line-height:1.25;text-align:justify;white-space:pre-wrap;overflow-wrap:anywhere;text-overflow:ellipsis",
		"border:2px solid #fff;border-left-color:red;border-radius:50%", "border-width:1px 2px;border-style:solid none",
		"background-position:right bottom;background-size:cover;background-repeat:repeat-x", "object-fit:scale-down;object-position:25% 2px;aspect-ratio:16 / 9",
		"background-position:bottom right;object-position:top", "background-size:50% auto", "line-height:normal;z-index:auto",
	}
	for _, source := range valid {
		if _, err := Parse(source, 1); err != nil {
			t.Errorf("%s: %v", source, err)
		}
	}
}

func TestVisualRejectsUnsupportedOrInvalidValues(t *testing.T) {
	invalid := []string{
		"opacity:1.1", "opacity:-1", "z-index:1.5", "z-index:32768", "font-weight:500", "font-size:0px", "line-height:0",
		"border:dashed", "border:-1px solid red", "border-radius:2px / 3px", "border-width:auto", "border-color:none",
		"color:rgb(300,0,0)", "color:rgba(0,0,0,2)", "color:#xyz", "color:rgb(1,2)", "color:rgb(1 2 3)",
		"display:flex", "position:fixed", "background-size:-1px auto", "background-position:left right", "object-position:top bottom",
		"aspect-ratio:0 / 1", "aspect-ratio:1 / 0", "background-image:url(file:///secret)", "background-image:url('data:image/png;base64,A;!important') !important",
		`font-family:"'Go'"`, `font-family:'Go"`, `font-family:Go,`, `font-family:"Go, sans-serif"`,
		"object-position:top 10px", "background-position:20px left",
	}
	for _, source := range invalid {
		if _, err := Parse(source, 1); err == nil {
			t.Errorf("accepted %s", source)
		}
	}
}

func TestColorsAreUnassociatedUntilCompositing(t *testing.T) {
	for input, want := range map[string]color.NRGBA{
		"#1234": {17, 34, 51, 68}, "#11223344": {17, 34, 51, 68},
		"rgba(255, 0, 0, 0.5)": {255, 0, 0, 128}, "transparent": {},
		"rgb(100%, 0%, 0%)": {255, 0, 0, 255}, "rebeccapurple": {102, 51, 153, 255},
	} {
		block, err := Parse("color:"+input, 1)
		got, _ := block.Get("color")
		if err != nil || got.Color != want {
			t.Errorf("%s: %+v %v want %+v", input, got.Color, err, want)
		}
	}
}

func TestBorderShorthandResetsAllSidesWithoutOverridingImportant(t *testing.T) {
	block, err := Parse("border-left-color:red!important;border:2px solid blue;border:none", 1)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := block.Get("border-left-color")
	right, _ := block.Get("border-right-color")
	style, _ := block.Get("border-top-style")
	if left.Color.R != 255 || right.Keyword != "currentcolor" || style.Keyword != "none" {
		t.Fatalf("incorrect reset: %+v %+v %+v", left, right, style)
	}
}
