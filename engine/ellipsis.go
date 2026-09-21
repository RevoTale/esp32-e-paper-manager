package engine

import (
	"strings"

	"github.com/tdewolff/canvas"
)

func (p *paragraph) elide() *canvas.Text {
	var source strings.Builder
	for _, run := range p.runs {
		source.WriteString(run.source)
	}
	ends := graphemeEnds(source.String())
	low, high := 0, len(ends)-1
	for low < high {
		middle := (low + high + 1) / 2
		candidate := p.typeset(p.prefix(ends[middle]))
		if candidate.Width <= p.width {
			low = middle
		} else {
			high = middle - 1
		}
	}
	return p.typeset(p.prefix(ends[low]))
}

func (p *paragraph) prefix(bytes int) []inlineRun {
	result := make([]inlineRun, 0, len(p.runs)+1)
	owner := p.root
	for _, run := range p.runs {
		if bytes == 0 {
			break
		}
		count := min(bytes, len(run.source))
		result = append(result, inlineRun{owner: run.owner, source: run.source[:count]})
		owner = run.owner
		bytes -= count
	}
	return append(result, inlineRun{owner: owner, source: "…"})
}
