package style

import (
	"errors"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func TestAspectRatioRejectsUnderflow(t *testing.T) {
	block, err := Parse("aspect-ratio:1e-320 / 8192", 13)
	if err == nil {
		value, _ := block.Get("aspect-ratio")
		t.Fatalf("positive ratio underflow accepted as %g", value.Number)
	}
	var diagnostic *renderdiag.Error
	if !errors.As(err, &diagnostic) || diagnostic.Code != renderdiag.InvalidValue {
		t.Fatalf("want invalid-value diagnostic, got %v", err)
	}
}

func TestScalarKeywordsIgnoreASCIICase(t *testing.T) {
	for _, tc := range []struct {
		property string
		keyword  string
	}{
		{"z-index", "auto"},
		{"line-height", "normal"},
		{"aspect-ratio", "auto"},
	} {
		t.Run(tc.property, func(t *testing.T) {
			block, err := Parse(tc.property+":"+strings.ToUpper(tc.keyword), 13)
			if err != nil {
				t.Fatalf("uppercase CSS keyword rejected: %v", err)
			}
			value, ok := block.Get(tc.property)
			if !ok || value.Kind != KeywordValue || value.Keyword != tc.keyword {
				t.Fatalf("want canonical keyword %q, got %+v", tc.keyword, value)
			}
		})
	}
}

func TestComputedFailureRetainsDeclarationLocation(t *testing.T) {
	const prefix = "color:red;"
	for _, declaration := range []string{"font-size:7px", "line-height:8192em"} {
		t.Run(declaration, func(t *testing.T) {
			source := prefix + declaration
			block, err := Parse(source, 13)
			if err != nil {
				t.Fatalf("specified value rejected before computation: %v", err)
			}
			_, err = Compute("div", block, nil, Metrics{ViewportWidth: 800, ViewportHeight: 480})
			var diagnostic *renderdiag.Error
			if !errors.As(err, &diagnostic) {
				t.Fatalf("computed rejection lost structured diagnostic: %v", err)
			}
			if diagnostic.Code != renderdiag.InvalidValue || diagnostic.Source != renderdiag.Inline || diagnostic.Element != 13 {
				t.Fatalf("computed rejection lost source identity: %+v", diagnostic)
			}
			if diagnostic.Start != uint32(len(prefix)) || diagnostic.End != uint32(len(source)) {
				t.Fatalf("want declaration span [%d,%d), got [%d,%d)", len(prefix), len(source), diagnostic.Start, diagnostic.End)
			}
		})
	}
}
