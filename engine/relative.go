package engine

import "github.com/RevoTale/esp32-e-paper-manager/engine/style"

// Relative offsets cannot borrow content-dependent height as a definite % base.
// WPT explicitly distinguishes auto/min-height from fixed/resolvable height:
// https://github.com/web-platform-tests/wpt/blob/dabe52e02fe620e75664f5904fb2b90fd95ac77f/css/css-position/position-relative-015.html
func (s *scene) relativeOffsets(e *element) (float64, float64, error) {
	area, definite, rtl := s.viewport, true, false
	// Inline ancestors do not establish this containing block. Direction belongs
	// to the containing block, not a dir override on the shifted child.
	// https://www.w3.org/TR/CSS22/visuren.html#relative-positioning
	for parent := e.parent; parent != nil; parent = parent.parent {
		if parent.css.Keyword("display") != "inline" {
			area, definite, rtl = parent.contentRect(), parent.box.DefiniteHeight, parent.direction == "rtl"
			break
		}
	}
	x, err := s.relativeAxis(e, "left", "right", area.Width, true, rtl)
	if err != nil {
		return 0, 0, err
	}
	y, err := s.relativeAxis(e, "top", "bottom", area.Height, definite, false)
	return x, y, err
}

func (s *scene) relativeAxis(e *element, start, end string, extent float64, definite, preferEnd bool) (float64, error) {
	a, hasStart, err := s.relativeInset(e, start, extent, definite)
	if err != nil {
		return 0, err
	}
	b, hasEnd, err := s.relativeInset(e, end, extent, definite)
	if err != nil {
		return 0, err
	}
	if hasEnd && preferEnd {
		return -b, nil
	}
	if hasStart {
		return a, nil
	}
	if hasEnd {
		return -b, nil
	}
	return 0, nil
}

func (s *scene) relativeInset(e *element, name string, extent float64, definite bool) (float64, bool, error) {
	if !definite && e.css.Length(name).Unit == style.Percent {
		return 0, false, nil
	}
	return s.inset(e, name, extent)
}
