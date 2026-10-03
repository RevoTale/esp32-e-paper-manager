package dom

import (
	"bytes"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func editFragment(node *Node, markup string) error {
	if err := preflight([]byte(markup)); err != nil {
		return err
	}
	if err := fragmentStructure(markup); err != nil {
		return err
	}
	context := &html.Node{Type: html.ElementNode, Data: node.Tag, DataAtom: atom.Lookup([]byte(node.Tag))}
	children, err := html.ParseFragment(strings.NewReader(markup), context)
	if err != nil {
		return editError()
	}
	b := builder{doc: Document{ByID: make(map[string]*Node)}}
	node.Children = nil
	for _, child := range children {
		if child.Type == html.CommentNode || child.Type == html.DoctypeNode {
			continue
		}
		converted, err := b.convert(child, node, 0)
		if err != nil {
			return err
		}
		node.Children = append(node.Children, converted)
	}
	return nil
}

func fragmentStructure(markup string) error {
	z := html.NewTokenizer(strings.NewReader(markup))
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			return nil
		}
		if kind == html.DoctypeToken {
			return editError()
		}
		if kind != html.StartTagToken && kind != html.EndTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		tag := z.Token().Data
		if tag == "html" || tag == "head" || tag == "body" {
			return editError()
		}
	}
}

// HTML5 parsing can reparent constructed subtrees (e.g. a div inside a p).
// Reject if the canonical intended tree is not stable through Render/Parse:
// https://pkg.go.dev/golang.org/x/net/html#Render
func roundTrip(doc *Document) ([]byte, error) {
	encoded, err := serialize(doc.Root)
	if err != nil {
		return nil, err
	}
	validated, err := Parse(encoded)
	if err != nil {
		return nil, err
	}
	again, err := serialize(validated.Root)
	if err != nil || !bytes.Equal(encoded, again) {
		return nil, editError()
	}
	return encoded, nil
}

func serialize(root *Node) ([]byte, error) {
	w := &boundedHTML{}
	if err := html.Render(w, htmlNode(root)); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func htmlNode(node *Node) *html.Node {
	result := &html.Node{Type: html.ElementNode, Data: node.Tag, DataAtom: atom.Lookup([]byte(node.Tag))}
	if node.Tag == "" {
		result.Type, result.Data = html.TextNode, node.Text
		if node.Parent == nil {
			result.Type = html.DocumentNode
		}
	}
	keys := make([]string, 0, len(node.Attrs))
	for key := range node.Attrs {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		result.Attr = append(result.Attr, html.Attribute{Key: key, Val: node.Attrs[key]})
	}
	for _, child := range node.Children {
		result.AppendChild(htmlNode(child))
	}
	return result
}

type boundedHTML struct{ bytes.Buffer }

func (w *boundedHTML) Write(data []byte) (int, error) {
	if len(data) > 32768-w.Len() {
		return 0, editError()
	}
	return w.Buffer.Write(data)
}

// html.Render uses WriteString/WriteByte when provided; keep all paths bounded.
func (w *boundedHTML) WriteString(data string) (int, error) { return w.Write([]byte(data)) }
func (w *boundedHTML) WriteByte(data byte) error            { _, err := w.Write([]byte{data}); return err }
