package style

import (
	"errors"
	"io"
	"strings"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

const MaxDeclarations = 4096

// Parse consumes an inline declaration list, not selectors or a stylesheet.
// Parser recovery never turns invalid input into a partially applied scene.
// https://pkg.go.dev/github.com/tdewolff/parse/v2/css#NewParser
func Parse(source string, element uint32) (Block, error) {
	if len(source) > 32768 {
		return Block{}, diagnostic(renderdiag.InputLimit, element, 0, len(source))
	}
	p := css.NewParser(parse.NewInputString(source), true)
	block := Block{values: make(map[string]declaration), element: element}
	for {
		start := p.Offset()
		kind, _, name := p.Next()
		if kind == css.ErrorGrammar {
			if errors.Is(p.Err(), io.EOF) && !p.HasParseError() {
				return block, nil
			}
			return Block{}, diagnostic(renderdiag.Syntax, element, start, p.Offset())
		}
		if kind == css.CommentGrammar {
			continue
		}
		location := renderdiag.Error{Source: renderdiag.Inline, Element: element, Start: uint32(start), End: uint32(p.Offset())}
		code := parseDeclaration(&block, kind, string(name), p.Values(), location)
		if code != 0 {
			return Block{}, diagnostic(code, element, start, p.Offset())
		}
	}
}

func parseDeclaration(block *Block, kind css.GrammarType, name string, tokens []css.Token, location renderdiag.Error) renderdiag.Code {
	if kind != css.DeclarationGrammar {
		return renderdiag.UnsupportedRule
	}
	block.count++
	if block.count > MaxDeclarations {
		return renderdiag.DeclarationLimit
	}
	if len(tokens) > 256 {
		return renderdiag.TokenLimit
	}
	raw, important := declarationValue(tokens)
	values, code := property(strings.ToLower(name), raw)
	if code == 0 {
		block.apply(values, important, location)
	}
	return code
}

func declarationValue(tokens []css.Token) (string, bool) {
	var builder strings.Builder
	for _, token := range tokens {
		builder.Write(token.Data)
	}
	value := strings.TrimSpace(builder.String())
	// The grammar supplies tokens, so ! inside a string or URL is not priority.
	compact := make([]css.Token, 0, len(tokens))
	for _, token := range tokens {
		if token.TokenType != css.WhitespaceToken {
			compact = append(compact, token)
		}
	}
	n := len(compact)
	if n >= 2 && compact[n-2].TokenType == css.DelimToken && string(compact[n-2].Data) == "!" &&
		compact[n-1].TokenType == css.IdentToken && strings.EqualFold(string(compact[n-1].Data), "important") {
		index := strings.LastIndex(value, "!")
		return strings.TrimSpace(value[:index]), true
	}
	return value, false
}

func diagnostic(code renderdiag.Code, element uint32, start, end int) error {
	return &renderdiag.Error{Code: code, Source: renderdiag.Inline, Element: element, Start: uint32(start), End: uint32(end)}
}
