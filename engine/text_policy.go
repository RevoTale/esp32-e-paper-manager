package engine

import (
	"strings"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
	"github.com/go-text/typesetting/segmenter"
	"github.com/tdewolff/canvas"
	canvastext "github.com/tdewolff/canvas/text"
)

type inlineRun struct {
	owner  *element
	source string
	object *canvas.Canvas
	face   *canvas.FontFace // Object-only identity; text uses its owner's face.
}

type forcedBreaks struct{}

func (forcedBreaks) Linebreak(items []canvastext.Item, _ float64) []canvastext.Break {
	var breaks []canvastext.Break
	width := 0.0
	for index, item := range items {
		if item.Type != canvastext.PenaltyType {
			width += item.Width
		} else if item.Penalty <= -canvastext.Infinity {
			breaks = append(breaks, canvastext.Break{Position: index, Width: width})
			width = 0
		}
	}
	return breaks
}

func (p *paragraph) prepareText() (*canvas.Text, error) {
	for i, run := range p.runs {
		if run.object == nil && run.owner.css.Keyword("overflow-wrap") == "anywhere" && wraps(run.owner) {
			p.runs[i].source = emergencyBreaks(run.source, p.face(run.owner), p.width)
		}
	}
	text := p.typeset(p.runs)
	if p.root.css.Keyword("text-overflow") == "ellipsis" {
		if !p.canElide(text) {
			return nil, p.root.reject("text-overflow")
		}
		if text.Width > p.width {
			// Ellipsis is paint-only: a hidden large-font run must still affect
			// its original line box and the following block's position.
			// https://www.w3.org/TR/css-overflow-4/#ellipsis-interaction
			text.WalkLines(func(_ float64, spans []canvas.TextSpan) {
				p.elidedMetrics[0], p.elidedMetrics[1] = p.lineMetrics(spans)
			})
			text = p.elide()
			p.scene.warn(p.root, renderdiag.Clipped)
		}
	}
	if text.OverflowsX {
		p.scene.warn(p.root, renderdiag.Clipped)
	}
	return text, nil
}

func (p *paragraph) canElide(text *canvas.Text) bool {
	return p.root.css.Keyword("white-space") == "nowrap" && p.root.css.Keyword("overflow") != "visible" && len(p.objects) == 0 && len(p.boundaries) == 0 && !strings.Contains(text.Text, "\n")
}

func wraps(e *element) bool {
	mode := e.css.Keyword("white-space")
	return mode != "nowrap" && mode != "pre"
}

func (p *paragraph) typeset(runs []inlineRun) *canvas.Text {
	text := canvas.NewRichText(p.face(p.root))
	for _, run := range runs {
		if run.object != nil {
			text.SetFace(run.face)
			text.WriteCanvas(run.object, canvas.Baseline)
		} else {
			text.WriteFace(p.face(run.owner), run.source)
		}
	}
	var breaker canvastext.Linebreaker = canvas.GreedyLinebreaker{}
	if !wraps(p.root) {
		breaker = forcedBreaks{}
	}
	return text.ToText(p.width, 0, textAlign(p.root), canvas.Top, &canvas.TextOptions{Linebreaker: breaker})
}

func graphemeEnds(text string) []int {
	var segment segmenter.Segmenter
	segment.InitWithString(text)
	iterator, offset := segment.GraphemeIterator(), 0
	ends := []int{0}
	for iterator.Next() {
		offset += len(string(iterator.Grapheme().Text))
		ends = append(ends, offset)
	}
	return ends
}

func emergencyBreaks(text string, face *canvas.FontFace, width float64) string {
	var result strings.Builder
	start := 0
	for index, r := range text {
		if cssSpace(r) {
			result.WriteString(breakWord(text[start:index], face, width))
			result.WriteRune(r)
			start = index + len(string(r))
		}
	}
	result.WriteString(breakWord(text[start:], face, width))
	return result.String()
}

func breakWord(word string, face *canvas.FontFace, width float64) string {
	if face.TextWidth(word) <= width {
		return word
	}
	ends, previous := graphemeEnds(word), 0
	var result strings.Builder
	for _, end := range ends[1:] {
		if previous != 0 {
			result.WriteRune('\u200B')
		}
		result.WriteString(word[previous:end])
		previous = end
	}
	return result.String()
}
