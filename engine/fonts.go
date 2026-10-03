package engine

import (
	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
	"github.com/tdewolff/canvas"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/gofont/goregular"
)

type fonts struct{ regular, mono *canvas.FontFamily }

// Each renderer owns its font/shaping objects. A renderer's admission gate
// serializes access; concurrent callers never race a dependency's glyph cache.
// https://pkg.go.dev/golang.org/x/image/font/gofont
func loadFonts() (fonts, error) {
	f := fonts{regular: canvas.NewFontFamily("Go"), mono: canvas.NewFontFamily("Go Mono")}
	for _, source := range []struct {
		family *canvas.FontFamily
		data   []byte
		style  canvas.FontStyle
	}{
		{f.regular, goregular.TTF, canvas.FontRegular}, {f.regular, gobold.TTF, canvas.FontBold},
		{f.regular, goitalic.TTF, canvas.FontItalic}, {f.regular, gobolditalic.TTF, canvas.FontBold | canvas.FontItalic},
		{f.mono, gomono.TTF, canvas.FontRegular}, {f.mono, gomonobold.TTF, canvas.FontBold},
		{f.mono, gomonoitalic.TTF, canvas.FontItalic}, {f.mono, gomonobolditalic.TTF, canvas.FontBold | canvas.FontItalic},
	} {
		if err := source.family.LoadFont(source.data, 0, source.style); err != nil {
			return fonts{}, err
		}
	}
	return f, nil
}

func (f fonts) face(computed style.Computed) *canvas.FontFace {
	family := f.regular
	if computed.Keyword("font-family") == "monospace" {
		family = f.mono
	}
	variant := canvas.FontRegular
	if computed.Keyword("font-weight") == "bold" {
		variant |= canvas.FontBold
	}
	if computed.Keyword("font-style") == "italic" {
		variant |= canvas.FontItalic
	}
	// Canvas accepts points and converts to mm. Our internal mm is one logical
	// pixel: cancel that conversion explicitly, independently of target DPI.
	return family.Face(computed.FontSize()*72/25.4, computed.Color("color"), variant)
}
