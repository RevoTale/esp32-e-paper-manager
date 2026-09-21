package engine

import (
	"math"
	"strings"

	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
)

func (s *scene) intrinsic(e *element) (geometry.Intrinsic, error) {
	result := geometry.Intrinsic{}
	line := 0.0
	face := s.fonts.face(e.css)
	for _, node := range e.node.Children {
		if node.Tag == "" {
			line += face.TextWidth(node.Text)
			for _, word := range strings.FieldsFunc(node.Text, cssSpace) {
				result.Minimum = math.Max(result.Minimum, face.TextWidth(word))
			}
		} else if child := s.elements[node]; inNormalFlow(child) {
			size, err := s.intrinsic(child)
			if err != nil {
				return geometry.Intrinsic{}, err
			}
			size, err = geometry.IntrinsicContribution(child.css, s.reference(s.viewport, false), size)
			if err != nil {
				return geometry.Intrinsic{}, child.reject("width")
			}
			if child.css.Keyword("display") == "block" {
				result.Preferred = math.Max(result.Preferred, math.Max(line, size.Preferred))
				line = 0
			} else {
				line += size.Preferred
			}
			result.Minimum = math.Max(result.Minimum, size.Minimum)
		}
	}
	result.Preferred = math.Max(result.Preferred, line)
	if img := s.assets.images[e.node]; img != nil {
		result.Minimum = float64(img.Bounds().Dx())
		result.Preferred = result.Minimum
	}
	return result, nil
}

func inNormalFlow(e *element) bool {
	return e != nil && !e.hidden && e.css.Keyword("position") != "absolute"
}

func cssSpace(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' }
