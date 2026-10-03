package engine

import (
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
	"github.com/tdewolff/canvas"
)

type glyphPaint struct {
	path  *canvas.Path
	style canvas.Style
	view  canvas.Matrix
	x, y  float64 // Paragraph origin in top-left logical coordinates.
}

type capturedPath struct {
	path  *canvas.Path
	style canvas.Style
	view  canvas.Matrix
}

type pathCapture struct {
	*canvas.Canvas
	paths []capturedPath
}

func (p *pathCapture) RenderPath(path *canvas.Path, style canvas.Style, view canvas.Matrix) {
	p.paths = append(p.paths, capturedPath{path, style, view})
}

func (p *paragraph) shape(x, y float64) (float64, error) {
	text, err := p.prepareText()
	if err != nil {
		return 0, err
	}
	capture := &pathCapture{Canvas: canvas.New(p.width, p.scene.viewport.Height)}
	// RenderTextTo exposes already-shaped glyph paths. Never reshape each word
	// at paint time: that would lose ligatures, bidi ordering and measured advances.
	// https://github.com/tdewolff/canvas/blob/dae8cd8e19a7/text.go#L1244
	text.RenderTextTo(capture, canvas.Identity, canvas.DPMM(1))
	index, height := 0, 0.0
	text.WalkLines(func(oldBaseline float64, spans []canvas.TextSpan) {
		p.fragments = make(map[*element]geometry.Rect)
		ascent, descent := p.lineMetrics(spans)
		baseline := height + ascent
		p.root.baseline, p.root.hasLine = y+baseline-p.root.y, true
		for _, span := range spans {
			if span.IsText() {
				p.placeGlyph(span, capture.paths[index], x, y, baseline, oldBaseline)
				index++
			} else {
				p.placeObjects(span, x, y+baseline)
			}
		}
		p.finishFragments()
		height += ascent + descent
	})
	if math.IsNaN(height) || height > 32768 {
		return 0, p.root.reject("line-height")
	}
	return height, p.scene.ctx.Err()
}

func textAlign(e *element) canvas.TextAlign {
	value := e.css.Keyword("text-align")
	if value == "start" && e.direction == "rtl" || value == "end" && e.direction != "rtl" {
		value = "right"
	}
	switch value {
	case "right":
		return canvas.Right
	case "center":
		return canvas.Center
	case "justify":
		return canvas.Justify
	}
	return canvas.Left
}

func lineHeight(e *element, face *canvas.FontFace) float64 {
	value := e.css.Get("line-height")
	if value.Kind == style.NumberValue {
		return value.Number * e.css.FontSize()
	}
	if value.Kind == style.LengthValue {
		return value.Length.Value
	}
	return face.Metrics().LineHeight
}

func (p *paragraph) lineMetrics(spans []canvas.TextSpan) (float64, float64) {
	face := p.face(p.root)
	metrics := face.Metrics()
	leading := (lineHeight(p.root, face) - metrics.Ascent - metrics.Descent) / 2
	ascent, descent := metrics.Ascent+leading, metrics.Descent+leading
	for _, span := range spans {
		m := span.Face.Metrics()
		leading = (lineHeight(p.owners[span.Face], span.Face) - m.Ascent - m.Descent) / 2
		ascent, descent = math.Max(ascent, m.Ascent+leading), math.Max(descent, m.Descent+leading)
		for _, object := range span.Objects {
			a, d := p.objectHeights(object, span.Face)
			ascent, descent = math.Max(ascent, a), math.Max(descent, d)
		}
	}
	return math.Max(ascent, p.elidedMetrics[0]), math.Max(descent, p.elidedMetrics[1])
}

func (p *paragraph) placeGlyph(span canvas.TextSpan, path capturedPath, x, y, baseline, oldBaseline float64) {
	e := p.owners[span.Face]
	for _, glyph := range span.Glyphs {
		if glyph.ID == 0 && glyph.Text != "\u200B" && glyph.Text != "\n" {
			p.scene.warn(e, renderdiag.MissingGlyph)
		}
	}
	// The Canvas glyph matrix uses upward Y; our stored paragraph origin uses
	// downward Y. Replace only its baseline, retaining shaped offsets and rotation.
	view := canvas.Identity.Translate(0, -baseline-oldBaseline).Mul(path.view)
	e.glyphs = append(e.glyphs, glyphPaint{path.path, path.style, view, x, y})
	m := span.Face.Metrics()
	fragment := geometry.Rect{X: x + span.X, Y: y + baseline - m.Ascent, Width: span.Width, Height: m.Ascent + m.Descent}
	p.addFragment(e, fragment)
}

func (p *paragraph) placeObjects(span canvas.TextSpan, x, baseline float64) {
	for _, object := range span.Objects {
		if edge, ok := p.boundaries[object.Canvas]; ok {
			p.placeBoundary(edge, x+span.X+object.X, baseline)
			continue
		}
		e := p.objects[object.Canvas]
		ascent, _ := p.objectHeights(object, span.Face)
		moveElement(e, x+span.X+object.X, baseline-ascent)
		m := p.face(e.parent).Metrics()
		p.addFragment(e.parent, geometry.Rect{X: x + span.X + object.X, Y: baseline - m.Ascent, Width: object.Width, Height: m.Ascent + m.Descent})
	}
}
