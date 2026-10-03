package engine

import (
	"github.com/RevoTale/esp32-e-paper-manager/engine/dom"
	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
)

// Adjacent vertical margins add in this bounded profile; they never collapse.
// CSS containing boxes, unlike transport regions, retain fractional pixels.
// https://www.w3.org/TR/CSS22/visudet.html#containing-block-details
func (s *scene) layoutBlock(e *element, ref geometry.Reference, x, y float64) error {
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if e.node.Tag == "img" {
		return s.layoutImage(e, ref, x, y)
	}
	resolve := geometry.ResolveWidth
	if e.css.Keyword("position") == "absolute" {
		resolve = geometry.ResolveAbsoluteWidth
	}
	intrinsic, err := s.intrinsic(e)
	if err != nil {
		return err
	}
	box, err := resolve(e.css, ref, intrinsic)
	if err != nil {
		return e.reject("width")
	}
	e.box, e.x, e.y = box, x+box.Margin.Left, y+box.Margin.Top
	if err := resolveHeight(e, 0); err != nil {
		return e.reject("height")
	}
	natural, err := s.flowWithMarker(e)
	if err != nil {
		return err
	}
	if err := resolveHeight(e, natural); err != nil {
		return e.reject("height")
	}
	if !e.borderRect().Valid() {
		return e.reject("position")
	}
	return nil
}

func resolveHeight(e *element, natural float64) error {
	if e.css.Keyword("position") == "absolute" {
		return e.box.ResolveAbsoluteHeight(e.css, natural)
	}
	return e.box.ResolveHeight(e.css, natural)
}

func (s *scene) flowChildren(e *element) (float64, error) {
	content, natural := e.contentRect(), 0.0
	var inline []*dom.Node
	flush := func() error {
		height, err := s.layoutInline(e, inline, content.X, content.Y+natural, content.Width)
		natural += height
		inline = nil
		return err
	}
	for _, node := range e.node.Children {
		child := s.elements[node]
		if child != nil && child.hidden {
			continue
		}
		if child != nil && child.css.Keyword("position") == "absolute" {
			s.absolute = append(s.absolute, child)
			child.x, child.y = content.X, content.Y+natural
			continue
		}
		if child == nil || child.css.Keyword("display") != "block" {
			inline = append(inline, node)
			continue
		}
		if err := flush(); err != nil {
			return 0, err
		}
		if err := s.layoutBlock(child, s.reference(content, e.box.DefiniteHeight), content.X, content.Y+natural); err != nil {
			return 0, err
		}
		e.adoptBaseline(child)
		natural += child.box.Margin.Vertical() + child.box.OuterHeight()
	}
	err := flush()
	return natural, err
}

func (s *scene) layoutImage(e *element, ref geometry.Reference, x, y float64) error {
	img := s.assets.images[e.node]
	box, err := geometry.ResolveReplaced(e.css, ref, float64(img.Bounds().Dx()), float64(img.Bounds().Dy()))
	if err != nil {
		return e.reject("width")
	}
	e.box, e.x, e.y = box, x+box.Margin.Left, y+box.Margin.Top
	return nil
}

func moveElement(e *element, x, y float64) {
	e.x, e.y = e.x+x, e.y+y
	for i := range e.glyphs {
		e.glyphs[i].x += x
		e.glyphs[i].y += y
	}
	for i := range e.fragments {
		e.fragments[i].X += x
		e.fragments[i].Y += y
	}
	for _, child := range e.children {
		moveElement(child, x, y)
	}
}
