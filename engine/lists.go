package engine

import (
	"strconv"

	"github.com/RevoTale/esp32-e-paper-manager/engine/dom"
)

// List markers are a small explicit user-agent profile: left-side bullets or
// decimal numbers. No CSS counter expressions, generated content or list images.
// https://www.w3.org/TR/CSS22/generate.html#lists
func listMarker(e *element) string {
	if e.node.Tag != "li" || e.parent == nil || e.css.Keyword("display") != "block" {
		return ""
	}
	if e.parent.node.Tag == "ul" {
		return "•"
	}
	if e.parent.node.Tag != "ol" {
		return ""
	}
	number := 0
	for _, sibling := range e.parent.children {
		if sibling.node.Tag == "li" && !sibling.hidden {
			number++
		}
		if sibling == e {
			break
		}
	}
	return strconv.Itoa(number) + "."
}

func (s *scene) layoutListMarker(e *element) (float64, error) {
	marker := listMarker(e)
	if marker == "" {
		return 0, nil
	}
	content := e.contentRect()
	width := s.fonts.face(e.css).TextWidth(marker)
	baseline, hasLine := e.baseline, e.hasLine
	height, err := s.layoutInline(e, []*dom.Node{{Text: marker}}, content.X-width-8, content.Y, width)
	// An outside marker must not replace the content's last in-flow baseline.
	e.baseline, e.hasLine = baseline, hasLine
	return height, err
}

func (s *scene) flowWithMarker(e *element) (float64, error) {
	natural, err := s.flowChildren(e)
	if err != nil {
		return 0, err
	}
	marker, err := s.layoutListMarker(e)
	return max(natural, marker), err
}
