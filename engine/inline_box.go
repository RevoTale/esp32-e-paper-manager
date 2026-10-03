package engine

import (
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
	"github.com/tdewolff/canvas"
)

type inlineBoundary struct {
	owner         *element
	start         bool
	width, margin float64
}

func (p *paragraph) appendInline(e *element) error {
	if e.css.Keyword("white-space") != p.root.css.Keyword("white-space") {
		return e.reject("white-space")
	}
	box, err := geometry.ResolveInline(e.css, p.scene.reference(geometry.Rect{Width: p.width}, false))
	if err != nil {
		return e.reject("padding")
	}
	// Canvas inline-object advances are non-negative. Reject this unsupported
	// combination rather than silently dropping a signed inline margin.
	if box.Margin.Left < 0 || box.Margin.Right < 0 {
		return e.reject("margin")
	}
	e.box = box
	if !inlineStartsLeft(e) {
		box.Padding.Left, box.Padding.Right = box.Padding.Right, box.Padding.Left
		box.Border.Left, box.Border.Right = box.Border.Right, box.Border.Left
		box.Margin.Left, box.Margin.Right = box.Margin.Right, box.Margin.Left
	}
	p.appendBoundary(e, true, box.Padding.Left+box.Border.Left, box.Margin.Left)
	for _, child := range e.node.Children {
		if err := p.appendNode(child, e); err != nil {
			return err
		}
	}
	p.appendBoundary(e, false, box.Padding.Right+box.Border.Right, box.Margin.Right)
	return nil
}

func (p *paragraph) appendBoundary(e *element, start bool, width, margin float64) {
	if width+margin == 0 {
		return
	}
	if start {
		p.flushSpace(e)
	}
	object := canvas.New(width+margin, 0)
	p.boundaries[object] = inlineBoundary{e, start, width, margin}
	p.writeObject(e, object)
}

func union(a, b geometry.Rect) geometry.Rect {
	x, y := math.Min(a.X, b.X), math.Min(a.Y, b.Y)
	return geometry.Rect{X: x, Y: y, Width: math.Max(a.Right(), b.Right()) - x, Height: math.Max(a.Bottom(), b.Bottom()) - y}
}

func (p *paragraph) addFragment(owner *element, area geometry.Rect) {
	for e := owner; e != p.root; e = e.parent {
		if old, ok := p.fragments[e]; ok {
			p.fragments[e] = union(old, area)
		} else {
			p.fragments[e] = area
		}
	}
}

func (p *paragraph) finishFragments() {
	for e, area := range p.fragments {
		area.Y -= e.box.Padding.Top + e.box.Border.Top
		area.Height += e.box.Padding.Vertical() + e.box.Border.Vertical()
		e.fragments = append(e.fragments, area)
	}
}

func (p *paragraph) placeBoundary(edge inlineBoundary, x, baseline float64) {
	m := p.face(edge.owner).Metrics()
	if edge.start == inlineStartsLeft(edge.owner) {
		x += edge.margin
	}
	p.addFragment(edge.owner, geometry.Rect{X: x, Y: baseline - m.Ascent, Width: edge.width, Height: m.Ascent + m.Descent})
}

// The parent's inline progression determines the unbroken physical edges,
// independently of a nested inline's own direction.
// https://www.w3.org/TR/css-break-3/#break-decoration
func inlineStartsLeft(e *element) bool { return e.parent == nil || e.parent.direction != "rtl" }
