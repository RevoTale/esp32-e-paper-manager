package dom

import (
	"strings"
	"unicode/utf8"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

// Edit targets an existing ID. Nil attribute values remove attributes; IDs are
// immutable. A batch resolves all targets against the same original document.
type Edit struct {
	ID         string
	Text, HTML *string
	Attributes map[string]*string
	Remove     bool
}

// Apply mutates only a private validated tree and returns canonical bounded HTML.
// No partial result escapes on any error, including HTML5 structural repairs.
func Apply(source []byte, edits []Edit) ([]byte, error) {
	if !boundedEdits(edits) {
		return nil, editError()
	}
	doc, err := Parse(source)
	if err != nil {
		return nil, err
	}
	targets, err := editTargets(doc, edits)
	if err != nil {
		return nil, err
	}
	for i, edit := range edits {
		if err := applyEdit(targets[i], edit); err != nil {
			return nil, err
		}
	}
	return roundTrip(doc)
}

func boundedEdits(edits []Edit) bool {
	if len(edits) == 0 || len(edits) > 64 {
		return false
	}
	remaining := 32768
	for _, edit := range edits {
		remaining -= len(edit.ID)
		for _, value := range []*string{edit.Text, edit.HTML} {
			if value != nil {
				remaining -= len(*value)
			}
		}
		for key, value := range edit.Attributes {
			remaining -= len(key)
			if value != nil {
				remaining -= len(*value)
			}
		}
		if remaining < 0 {
			return false
		}
	}
	return true
}

func editTargets(doc *Document, edits []Edit) ([]*Node, error) {
	selected := make(map[*Node]bool)
	targets := make([]*Node, len(edits))
	for i, edit := range edits {
		node := doc.ByID[edit.ID]
		if node == nil || selected[node] || !validOperation(edit) {
			return nil, editError()
		}
		if strings.Contains(" html head body ", " "+node.Tag+" ") && structuralEdit(edit) {
			return nil, editError()
		}
		selected[node], targets[i] = true, node
	}
	for _, node := range targets {
		for parent := node.Parent; parent != nil; parent = parent.Parent {
			if selected[parent] {
				return nil, editError()
			}
		}
	}
	return targets, nil
}

func structuralEdit(edit Edit) bool { return edit.Remove || edit.Text != nil || edit.HTML != nil }

func validOperation(edit Edit) bool {
	content := 0
	for _, text := range []*string{edit.Text, edit.HTML} {
		if text == nil {
			continue
		}
		if len(*text) > 32768 || !utf8.ValidString(*text) {
			return false
		}
		content++
	}
	if edit.Remove {
		return content == 0 && len(edit.Attributes) == 0
	}
	return content <= 1 && (content != 0 || len(edit.Attributes) != 0) && len(edit.Attributes) <= 32
}

func applyEdit(node *Node, edit Edit) error {
	if edit.Remove {
		for i, child := range node.Parent.Children {
			if child == node {
				node.Parent.Children = append(node.Parent.Children[:i], node.Parent.Children[i+1:]...)
				break
			}
		}
		return nil
	}
	if err := editAttributes(node, edit.Attributes); err != nil {
		return err
	}
	if edit.Text == nil && edit.HTML == nil {
		return nil
	}
	if strings.Contains(" img br hr meta ", " "+node.Tag+" ") {
		return editError()
	}
	if edit.Text != nil {
		node.Children = []*Node{{Text: *edit.Text, Parent: node}}
		return nil
	}
	return editFragment(node, *edit.HTML)
}

func editAttributes(node *Node, attrs map[string]*string) error {
	for key, value := range attrs {
		// Attribute keys are not escaped by html.Render. Only exact allowlisted
		// names can be inserted, before any serialization or reparsing.
		// https://pkg.go.dev/golang.org/x/net/html#Attribute
		if key == "id" || !allowedAttribute(node.Tag, key) {
			return editError()
		}
		if value == nil {
			delete(node.Attrs, key)
			continue
		}
		if len(*value) > 32768 || !utf8.ValidString(*value) {
			return editError()
		}
		node.Attrs[key] = *value
	}
	return nil
}

func editError() error { return rejection(renderdiag.InvalidDocument, 0) }
