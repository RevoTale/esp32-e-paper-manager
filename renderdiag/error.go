// Package renderdiag defines source-free document rejection diagnostics shared
// by render adapters and the manager. It has no renderer or device dependency.
package renderdiag

import (
	"errors"
	"fmt"
)

var ErrRejected = errors.New("render: document rejected")

// Code values 1–10 match the stable local BZE1 CSS diagnostic contract.
type Code uint16

const (
	InputLimit       Code = 1
	TokenLimit       Code = 2
	NestingLimit     Code = 3
	DeclarationLimit Code = 4
	NonFiniteNumber  Code = 5
	Syntax           Code = 6
	UnknownProperty  Code = 7
	InvalidValue     Code = 8
	InvalidSelector  Code = 9
	UnsupportedRule  Code = 10
	// InvalidDocument is adapter-local, never accepted as a BZE1 wire code.
	InvalidDocument Code = 11
)

type Source string

const (
	Inline     Source = "inline"
	Stylesheet Source = "stylesheet"
	HTML       Source = "html"
)

// Error describes one CSS source in canonical document order, including implied
// html/head/body elements. Start/End are decoded CSS UTF-8 byte offsets, not HTML
// offsets. Never retain source text, URLs or an upstream parser error here.
type Error struct {
	Code    Code   `json:"code"`
	Source  Source `json:"source"`
	Element uint32 `json:"element"`
	Start   uint32 `json:"start"`
	End     uint32 `json:"end"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("render: rejected %s code=%d element=%d bytes=%d..%d", e.Source, e.Code, e.Element, e.Start, e.End)
}

func (e *Error) Unwrap() error { return ErrRejected }
