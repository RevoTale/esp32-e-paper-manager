package engine

import (
	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

func (s *scene) containingBox(e *element) geometry.Rect {
	for parent := e.parent; parent != nil; parent = parent.parent {
		if parent.css.Keyword("position") != "static" {
			if parent.css.Keyword("display") == "inline" && len(parent.fragments) > 0 {
				area := parent.fragments[0]
				for _, fragment := range parent.fragments[1:] {
					area = union(area, fragment)
				}
				b := parent.box.Border
				return geometry.Rect{X: area.X + b.Left, Y: area.Y + b.Top, Width: area.Width - b.Horizontal(), Height: area.Height - b.Vertical()}
			}
			return parent.paddingRect()
		}
	}
	return s.viewport
}

func (s *scene) inset(e *element, name string, extent float64) (float64, bool, error) {
	length := e.css.Length(name)
	if length.Unit == style.Auto {
		return 0, false, nil
	}
	value, err := length.Resolve(style.Metrics{Containing: extent, Font: e.css.FontSize(), RootFont: e.css.RootFont(), ViewportWidth: s.viewport.Width, ViewportHeight: s.viewport.Height})
	if err != nil {
		return 0, true, e.reject(name)
	}
	return value, true, nil
}

func (s *scene) positionAxis(e *element, start, end string, origin, extent, outer, margin, endMargin, fallback float64) (float64, error) {
	a, hasStart, err := s.inset(e, start, extent)
	if err != nil {
		return 0, err
	}
	b, hasEnd, err := s.inset(e, end, extent)
	if err != nil {
		return 0, err
	}
	if hasStart && hasEnd {
		margin, endMargin = s.absoluteMargins(e, start, end, extent-a-b-outer, margin, endMargin)
		if start == "left" && s.absoluteRTL(e) {
			return origin + extent - b - outer - endMargin, nil
		}
	}
	if hasStart {
		return origin + a + margin, nil
	}
	if hasEnd {
		return origin + extent - b - outer - endMargin, nil
	}
	return fallback, nil
}

func (s *scene) layoutAbsolute(e *element) error {
	area := s.containingBox(e)
	x, y := e.x, e.y // Static-position fallback captured during normal flow.
	if err := s.layoutBlock(e, s.reference(area, true), x, y); err != nil {
		return err
	}
	x, err := s.positionAxis(e, "left", "right", area.X, area.Width, e.box.OuterWidth(), e.box.Margin.Left, e.box.Margin.Right, e.x)
	if err != nil {
		return err
	}
	y, err = s.positionAxis(e, "top", "bottom", area.Y, area.Height, e.box.OuterHeight(), e.box.Margin.Top, e.box.Margin.Bottom, e.y)
	if err != nil {
		return err
	}
	moveElement(e, x-e.x, y-e.y)
	return nil
}

func (s *scene) applyRelative(e *element) error {
	if e.css.Keyword("position") == "relative" && !e.hidden {
		x, y, err := s.relativeOffsets(e)
		if err != nil {
			return err
		}
		moveElement(e, x, y)
	}
	if !e.borderRect().Valid() {
		return e.reject("position")
	}
	for _, child := range e.children {
		if err := s.applyRelative(child); err != nil {
			return err
		}
	}
	return nil
}
