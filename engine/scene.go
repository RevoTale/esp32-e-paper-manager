package engine

import (
	"context"
	"strconv"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine/dom"
	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

type element struct {
	node      *dom.Node
	css       style.Computed
	parent    *element
	children  []*element
	box       geometry.Box
	x, y      float64
	hidden    bool
	direction string
	glyphs    []glyphPaint
	fragments []geometry.Rect
	baseline  float64 // Last in-flow line baseline, relative to the border top.
	hasLine   bool
}

func (e *element) borderRect() geometry.Rect {
	return geometry.Rect{X: e.x, Y: e.y, Width: e.box.OuterWidth(), Height: e.box.OuterHeight()}
}

func (e *element) paddingRect() geometry.Rect {
	b := e.box.Border
	return geometry.Rect{X: e.x + b.Left, Y: e.y + b.Top, Width: e.box.OuterWidth() - b.Horizontal(), Height: e.box.OuterHeight() - b.Vertical()}
}

func (e *element) contentRect() geometry.Rect {
	b, p := e.box.Border, e.box.Padding
	return geometry.Rect{X: e.x + b.Left + p.Left, Y: e.y + b.Top + p.Top, Width: e.box.ContentWidth, Height: e.box.ContentHeight}
}

func (e *element) reject(property string) error {
	return e.node.Style.Error(property, renderdiag.InvalidValue)
}

type scene struct {
	ctx      context.Context
	viewport geometry.Rect
	document *dom.Document
	assets   *assets
	fonts    fonts
	root     *element
	elements map[*dom.Node]*element
	absolute []*element
	warnings []renderdiag.Warning
}

func (s *scene) warn(e *element, code renderdiag.WarningCode) {
	warning := renderdiag.Warning{Code: code, Element: e.node.Ordinal}
	for _, existing := range s.warnings {
		if existing == warning {
			return
		}
	}
	s.warnings = append(s.warnings, warning)
}

func layoutScene(ctx context.Context, size display.Size, doc *dom.Document, a *assets, f fonts) (*scene, error) {
	s := &scene{ctx: ctx, viewport: geometry.Rect{Width: float64(size.Width), Height: float64(size.Height)}, document: doc, assets: a, fonts: f, elements: make(map[*dom.Node]*element)}
	var err error
	// The synthetic parser document is not a CSS box. HTML receives the definite
	// viewport directly, including its percentage-height containing block.
	s.root, err = s.build(doc.Root.Children[0], nil)
	if err != nil {
		return nil, err
	}
	ref := s.reference(s.viewport, true)
	if err = s.layoutBlock(s.root, ref, 0, 0); err != nil {
		return nil, err
	}
	for index := 0; index < len(s.absolute); index++ {
		if err = s.layoutAbsolute(s.absolute[index]); err != nil {
			return nil, err
		}
	}
	if err = s.applyRelative(s.root); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *scene) reference(area geometry.Rect, definite bool) geometry.Reference {
	return geometry.Reference{Width: area.Width, Height: area.Height, DefiniteHeight: definite, ViewportWidth: s.viewport.Width, ViewportHeight: s.viewport.Height}
}

func (s *scene) build(node *dom.Node, parent *element) (*element, error) {
	if err := s.ctx.Err(); err != nil {
		return nil, err
	}
	e := &element{node: node, parent: parent, direction: "ltr"}
	var inherited *style.Computed
	if parent != nil {
		inherited, e.direction, e.hidden = &parent.css, parent.direction, parent.hidden
	}
	block, err := elementStyle(node)
	if err != nil {
		return nil, err
	}
	css, err := style.Compute(node.Tag, block, inherited, style.Metrics{ViewportWidth: s.viewport.Width, ViewportHeight: s.viewport.Height})
	if err != nil {
		return nil, err
	}
	e.css = css
	e.hidden = e.hidden || css.Keyword("display") == "none" || metadata(node.Tag)
	if direction := node.Attrs["dir"]; direction != "" {
		e.direction = direction
	}
	s.elements[node] = e
	if err := s.buildChildren(e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *scene) buildChildren(e *element) error {
	for _, child := range e.node.Children {
		if child.Tag == "" {
			continue
		}
		built, err := s.build(child, e)
		if err != nil {
			return err
		}
		e.children = append(e.children, built)
	}
	return nil
}

func metadata(tag string) bool { return tag == "head" || tag == "title" || tag == "meta" }

func elementStyle(node *dom.Node) (style.Block, error) {
	if node.Tag != "img" {
		return node.Style, nil
	}
	// DOM validation already restricts presentational dimensions to 1..8192.
	width, _ := strconv.Atoi(node.Attrs["width"])
	height, _ := strconv.Atoi(node.Attrs["height"])
	return node.Style.WithDimensionHints(float64(width), float64(height))
}
