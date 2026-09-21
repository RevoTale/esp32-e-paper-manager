package engine

import "github.com/tdewolff/canvas"

func (e *element) adoptBaseline(child *element) {
	if child.hasLine {
		e.baseline, e.hasLine = child.y-e.y+child.baseline, true
	}
}

func (p *paragraph) objectHeights(object canvas.TextSpanObject, face *canvas.FontFace) (float64, float64) {
	e := p.objects[object.Canvas]
	if e == nil || e.node.Tag == "img" || !e.hasLine || e.css.Keyword("overflow") != "visible" {
		return object.Heights(face)
	}
	// Canvas supplies atomic widths and line breaks, but its Baseline object
	// alignment always uses the bottom edge. CSS inline-blocks instead export
	// their last in-flow line, unless overflow is non-visible or no line exists.
	// https://www.w3.org/TR/CSS22/visudet.html#leading
	ascent := e.box.Margin.Top + e.baseline
	return ascent, object.Height - ascent
}
