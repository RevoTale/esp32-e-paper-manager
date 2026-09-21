package engine

import "github.com/RevoTale/esp32-e-paper-manager/engine/style"

// Solve the margin-box equation only when both insets are known, after final
// min/max-constrained dimensions. The vertical axis permits equal negative
// auto margins; horizontal negative space belongs to the direction's end side.
// https://www.w3.org/TR/CSS22/visudet.html#abs-non-replaced-width
// https://www.w3.org/TR/CSS22/visudet.html#abs-non-replaced-height
func (s *scene) absoluteMargins(e *element, start, end string, space, before, after float64) (float64, float64) {
	autoBefore := e.css.Length("margin-"+start).Unit == style.Auto
	autoAfter := e.css.Length("margin-"+end).Unit == style.Auto
	remaining := space - before - after
	switch {
	case autoBefore && autoAfter:
		before, after = remaining/2, remaining/2
		if start == "left" && remaining < 0 {
			before, after = 0, remaining
			if s.absoluteRTL(e) {
				before, after = remaining, 0
			}
		}
	case autoBefore:
		before = remaining
	case autoAfter:
		after = remaining
	}
	if start == "left" {
		e.box.Margin.Left, e.box.Margin.Right = before, after
	} else {
		e.box.Margin.Top, e.box.Margin.Bottom = before, after
	}
	return before, after
}

func (s *scene) absoluteRTL(e *element) bool {
	for parent := e.parent; parent != nil; parent = parent.parent {
		if parent.css.Keyword("position") != "static" {
			return parent.direction == "rtl"
		}
	}
	return s.root.direction == "rtl"
}
