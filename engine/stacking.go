package engine

import (
	"sort"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

type paintEntry struct {
	element *element
	phase   int
	z       float64
	context bool
	atomic  bool
	box     bool
	content bool
}

func stackingContext(e *element) bool {
	return e.css.Number("opacity") < 1 || e.css.Keyword("position") != "static" && e.css.Get("z-index").Kind == style.NumberValue
}

// Contexts are atomic. A child's z-index is compared only with entries in its
// nearest context, never with arbitrary global descendants.
// https://www.w3.org/TR/CSS22/zindex.html
func contextEntries(root *element, includeContexts bool) []paintEntry {
	entries := []paintEntry{{element: root, phase: 2, content: true}}
	for _, child := range root.children {
		collectPaint(child, false, includeContexts, &entries)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].phase != entries[j].phase {
			return entries[i].phase < entries[j].phase
		}
		return entries[i].z < entries[j].z
	})
	return entries
}

func collectPaint(e *element, positioned, includeContexts bool, entries *[]paintEntry) {
	if e.hidden {
		return
	}
	if stackingContext(e) {
		if includeContexts {
			*entries = append(*entries, contextEntry(e))
		}
		return
	}
	positioned = positioned || e.css.Keyword("position") != "static"
	if e.css.Keyword("display") == "inline-block" {
		phase := 2
		if positioned {
			phase = 3
		}
		*entries = append(*entries, paintEntry{element: e, phase: phase, atomic: true})
		if includeContexts {
			collectEscaping(e, entries)
		}
		return
	}
	collectNormal(e, positioned, entries)
	for _, child := range e.children {
		collectPaint(child, positioned, includeContexts, entries)
	}
}

func collectNormal(e *element, positioned bool, entries *[]paintEntry) {
	if positioned {
		*entries = append(*entries, paintEntry{element: e, phase: 3, box: true, content: true})
	} else {
		phase := 1
		if e.css.Keyword("display") != "block" {
			phase = 2
		}
		*entries = append(*entries, paintEntry{element: e, phase: phase, box: true}, paintEntry{element: e, phase: 2, content: true})
	}

}

func contextEntry(e *element) paintEntry {
	phase, z := 3, 0.0
	if e.css.Keyword("position") != "static" {
		z = e.css.Number("z-index")
	}
	if z < 0 {
		phase = 0
	} else if z > 0 {
		phase = 4
	}
	return paintEntry{element: e, phase: phase, z: z, context: true}
}

func collectEscaping(root *element, entries *[]paintEntry) {
	for _, e := range root.children {
		if e.hidden {
			continue
		}
		if stackingContext(e) {
			*entries = append(*entries, contextEntry(e))
		} else {
			collectEscaping(e, entries)
		}
	}
}
