package dom

import (
	"bytes"
	"io"
	"slices"
	"strings"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
	"github.com/tdewolff/parse/v2"
	htmllex "github.com/tdewolff/parse/v2/html"
	"golang.org/x/net/html"
)

// preflight is a lexical work/active-input guard, not another DOM. HTML5 can
// discard repeated head/body tokens; validate those attributes before discard.
func preflight(source []byte) error {
	z := html.NewTokenizer(bytes.NewReader(source))
	for work := 0; ; work++ {
		if work > 4096 {
			return rejection(renderdiag.TokenLimit, 0)
		}
		kind := z.Next()
		if kind == html.ErrorToken {
			if z.Err() == io.EOF {
				return nil
			}
			return rejection(renderdiag.Syntax, 0)
		}
		if kind == html.StartTagToken || kind == html.SelfClosingTagToken {
			if err := rawAttributeGuard(z.Raw()); err != nil {
				return err
			}
			token := z.Token()
			if err := validateElement(token.Data, token.Attr); err != nil {
				return err
			}
		}
	}
}

// x/net/html v0.58.0 discards duplicate attributes during tokenization. Inspect
// the bounded raw start token before that normalization; never build a second DOM.
// https://cs.opensource.google/go/x/net/+/refs/tags/v0.58.0:html/token.go;l=903
func rawAttributeGuard(raw []byte) error {
	lexer := htmllex.NewLexer(parse.NewInputBytes(raw))
	seen := make(map[string]bool)
	for {
		kind, _ := lexer.Next()
		if kind == htmllex.ErrorToken {
			return nil
		}
		if kind != htmllex.AttributeToken {
			continue
		}
		key := strings.ToLower(string(lexer.AttrKey()))
		if seen[key] || len(seen) >= 32 {
			return rejection(renderdiag.InvalidDocument, 0)
		}
		seen[key] = true
	}
}

func validateElement(tag string, attrs []html.Attribute) error {
	const tags = " html head body title meta div section article main header footer aside span p h1 h2 h3 h4 h5 h6 strong b em i small br hr ul ol li figure figcaption img pre code "
	if !strings.Contains(tags, " "+tag+" ") || len(attrs) > 32 {
		return rejection(renderdiag.InvalidDocument, 0)
	}
	seen := make(map[string]bool, len(attrs))
	for _, attr := range attrs {
		if attr.Namespace != "" || seen[attr.Key] || !allowedAttribute(tag, attr.Key) {
			return rejection(renderdiag.InvalidDocument, 0)
		}
		seen[attr.Key] = true
		if !validAttributeValue(attr) {
			return rejection(renderdiag.InvalidDocument, 0)
		}
	}
	return nil
}

func allowedAttribute(tag, name string) bool {
	if slices.Contains([]string{"id", "class", "style", "lang", "dir", "title", "data-epaper-bitmap"}, name) {
		return true
	}
	return tag == "img" && slices.Contains([]string{"src", "alt", "width", "height"}, name) || tag == "meta" && name == "charset"
}

func validAttributeValue(attr html.Attribute) bool {
	switch attr.Key {
	case "dir":
		return attr.Val == "ltr" || attr.Val == "rtl"
	case "charset":
		return strings.EqualFold(attr.Val, "utf-8")
	case "src":
		return strings.HasPrefix(attr.Val, "data:image/png;base64,") || strings.HasPrefix(attr.Val, "data:image/jpeg;base64,")
	case "width", "height":
		return htmlDimension(attr.Val)
	default:
		return true
	}
}

func htmlDimension(value string) bool {
	if len(value) == 0 || len(value) > 4 {
		return false
	}
	number := 0
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
		number = number*10 + int(digit-'0')
	}
	return number > 0 && number <= 8192
}
