package style

import (
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

func familyValue(raw string) (Value, error) {
	lexer := css.NewLexer(parse.NewInputString(raw))
	first, expectFamily := "", true
	for {
		kind, data := lexer.Next()
		if kind == css.ErrorToken {
			break
		}
		if kind == css.WhitespaceToken {
			continue
		}
		if kind == css.CommaToken && !expectFamily {
			expectFamily = true
			continue
		}
		family, err := familyToken(kind, data)
		if err != nil || !expectFamily {
			return Value{}, ErrValue
		}
		if first == "" {
			first = family
		}
		expectFamily = false
	}
	if expectFamily {
		return Value{}, ErrValue
	}
	return Value{Kind: KeywordValue, Keyword: first}, nil
}

func familyToken(kind css.TokenType, data []byte) (string, error) {
	value := string(data)
	if kind == css.StringToken {
		if len(value) < 2 || value[len(value)-1] != value[0] {
			return "", ErrValue
		}
		value = value[1 : len(value)-1]
	} else if kind != css.IdentToken {
		return "", ErrValue
	}
	keyword, err := keywordValue(strings.ToLower(value), "go sans-serif monospace")
	return keyword.Keyword, err
}
