package style

import (
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func TestDeclarationOrderAndShorthandPriority(t *testing.T) {
	block, err := Parse("margin:1px 2px !important; margin-left:9px; margin-top:3px!important; WIDTH:50%;width:25vw", 7)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]Length{
		"margin-top": {Value: 3}, "margin-right": {Value: 2},
		"margin-bottom": {Value: 1}, "margin-left": {Value: 2},
		"width": {Value: 25, Unit: VW},
	} {
		got, ok := block.Get(name)
		if !ok || got.Length != want {
			t.Errorf("%s: %+v, %v; want %+v", name, got, ok, want)
		}
	}
}

func TestStyleBoundaryDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		input string
		code  renderdiag.Code
	}{
		{"unknown:1px", renderdiag.UnknownProperty},
		{"width:banana", renderdiag.InvalidValue},
		{"padding:-1px", renderdiag.InvalidValue},
		{"padding:auto", renderdiag.InvalidValue},
		{"width:none", renderdiag.InvalidValue},
		{"width:-2px", renderdiag.InvalidValue},
		{"width:1px!important!important", renderdiag.InvalidValue},
		{"width", renderdiag.Syntax},
		{"@import 'private';", renderdiag.UnsupportedRule},
		{"--secret: 'sensitive';", renderdiag.UnsupportedRule},
		{"width:calc(5px)", renderdiag.InvalidValue},
	} {
		t.Run(tc.input, func(t *testing.T) {
			_, err := Parse(tc.input, 7)
			var diag *renderdiag.Error
			if !errors.As(err, &diag) || diag.Code != tc.code || diag.Element != 7 {
				t.Fatalf("got %v; want code %d at element 7", err, tc.code)
			}
		})
	}
}

func TestDeclarationParserHandlesCommentsAndSemicolonsInValues(t *testing.T) {
	block, err := Parse("/* ignored */ width : 2px; margin:0;", 1)
	if err != nil || block.Count() != 2 {
		t.Fatalf("%+v %v", block, err)
	}
	_, err = Parse("background-image:url('data:image/png;base64,AAAA');width:2px", 1)
	if err != nil {
		t.Fatal(err)
	}
}
