package style

import (
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

type componentBuilder struct {
	values  []string
	current strings.Builder
	depth   int
}

// components splits grammar tokens only at top-level whitespace or slash.
// Quoted URLs and functions stay intact; no semicolon or slash string-splitting.
func components(raw string) []string {
	lexer := css.NewLexer(parse.NewInputString(raw))
	var result componentBuilder
	for {
		kind, data := lexer.Next()
		if kind == css.ErrorToken {
			break
		}
		result.accept(kind, data)
	}
	result.flush()
	return result.values
}

func (b *componentBuilder) accept(kind css.TokenType, data []byte) {
	slash := kind == css.DelimToken && string(data) == "/"
	if b.depth == 0 && (kind == css.WhitespaceToken || slash) {
		b.flush()
		if slash {
			b.values = append(b.values, "/")
		}
		return
	}
	switch kind {
	case css.FunctionToken, css.LeftParenthesisToken:
		b.depth++
	case css.RightParenthesisToken:
		b.depth--
	}
	b.current.Write(data)
}

func (b *componentBuilder) flush() {
	if b.current.Len() > 0 {
		b.values = append(b.values, b.current.String())
		b.current.Reset()
	}
}
