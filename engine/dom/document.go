// Package dom owns the engine's single validated HTML5 tree. It performs no I/O
// beyond reading its supplied bytes and never fetches assets or ambient fonts.
package dom

import (
	"bytes"
	"unicode/utf8"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
	"golang.org/x/net/html"
)

type Node struct {
	Tag, Text string
	Ordinal   uint32
	Attrs     map[string]string
	Style     style.Block
	Parent    *Node
	Children  []*Node
}

type Document struct {
	Root *Node
	ByID map[string]*Node
}

type builder struct {
	doc          Document
	ordinal      uint32
	nodes        int
	declarations int
}

func Parse(source []byte) (*Document, error) {
	if len(source) > 32768 || !utf8.Valid(source) {
		return nil, rejection(renderdiag.InputLimit, 0)
	}
	if err := preflight(source); err != nil {
		return nil, err
	}
	// HTML5 repairs optional end tags and inserts implied html/head/body nodes.
	// Validation, style ownership and later layout all use this exact tree.
	// https://pkg.go.dev/golang.org/x/net/html#Parse
	root, err := html.Parse(bytes.NewReader(source))
	if err != nil {
		return nil, rejection(renderdiag.InvalidDocument, 0)
	}
	b := builder{doc: Document{ByID: make(map[string]*Node)}}
	b.doc.Root, err = b.convert(root, nil, 0)
	if err != nil {
		return nil, err
	}
	return &b.doc, nil
}

func (b *builder) convert(source *html.Node, parent *Node, depth int) (*Node, error) {
	b.nodes++
	if depth > 32 || b.nodes > 1024 {
		return nil, rejection(renderdiag.NestingLimit, b.ordinal)
	}
	node := &Node{Parent: parent, Attrs: make(map[string]string)}
	switch source.Type {
	case html.ElementNode:
		if err := b.element(node, source); err != nil {
			return nil, err
		}
	case html.TextNode:
		node.Text = source.Data
	}
	for child := source.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.CommentNode || child.Type == html.DoctypeNode {
			continue
		}
		converted, err := b.convert(child, node, depth+1)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, converted)
	}
	return node, nil
}

func (b *builder) element(node *Node, source *html.Node) error {
	b.ordinal++
	node.Tag, node.Ordinal = source.Data, b.ordinal
	if source.Namespace != "" {
		return rejection(renderdiag.InvalidDocument, b.ordinal)
	}
	if err := validateElement(node.Tag, source.Attr); err != nil {
		return err
	}
	for _, attr := range source.Attr {
		node.Attrs[attr.Key] = attr.Val
	}
	if id, exists := node.Attrs["id"]; exists {
		if id == "" || b.doc.ByID[id] != nil {
			return rejection(renderdiag.InvalidDocument, b.ordinal)
		}
		b.doc.ByID[id] = node
	}
	var err error
	node.Style, err = style.Parse(node.Attrs["style"], b.ordinal)
	if err != nil {
		return err
	}
	b.declarations += node.Style.Count()
	if b.declarations > style.MaxDeclarations {
		return rejection(renderdiag.DeclarationLimit, b.ordinal)
	}
	return nil
}

func rejection(code renderdiag.Code, ordinal uint32) error {
	return &renderdiag.Error{Code: code, Source: renderdiag.HTML, Element: ordinal}
}
