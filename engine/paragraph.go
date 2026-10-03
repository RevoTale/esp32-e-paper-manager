package engine

import (
	"strings"

	"github.com/RevoTale/esp32-e-paper-manager/engine/dom"
	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
	"github.com/tdewolff/canvas"
	canvastext "github.com/tdewolff/canvas/text"
)

type paragraph struct {
	scene         *scene
	root          *element
	text          *canvas.RichText
	faces         map[*element]*canvas.FontFace
	owners        map[*canvas.FontFace]*element
	objects       map[*canvas.Canvas]*element
	boundaries    map[*canvas.Canvas]inlineBoundary
	fragments     map[*element]geometry.Rect
	width         float64
	x, y          float64 // Static-position estimate for out-of-flow inline descendants.
	pending       bool
	runs          []inlineRun
	elidedMetrics [2]float64
}

func (s *scene) layoutInline(root *element, nodes []*dom.Node, x, y, width float64) (float64, error) {
	if len(nodes) == 0 {
		return 0, nil
	}
	p := &paragraph{scene: s, root: root, faces: make(map[*element]*canvas.FontFace), owners: make(map[*canvas.FontFace]*element), objects: make(map[*canvas.Canvas]*element), width: width}
	p.x, p.y = x, y
	p.boundaries = make(map[*canvas.Canvas]inlineBoundary)
	p.text = canvas.NewRichText(p.face(root))
	for _, node := range nodes {
		if err := p.appendNode(node, root); err != nil {
			return 0, err
		}
	}
	if p.text.Len() == 0 {
		return 0, nil
	}
	return p.shape(x, y)
}

func (p *paragraph) face(e *element) *canvas.FontFace {
	if face := p.faces[e]; face != nil {
		return face
	}
	face := p.scene.fonts.face(e.css)
	if e.direction == "rtl" {
		face.Direction = canvastext.RightToLeft
	}
	p.faces[e], p.owners[face] = face, e
	return face
}

func (p *paragraph) appendNode(node *dom.Node, owner *element) error {
	if node.Tag == "" {
		return p.appendText(node.Text, owner)
	}
	e := p.scene.elements[node]
	if e.hidden {
		return nil
	}
	if e.css.Keyword("position") == "absolute" {
		e.x, e.y = p.x, p.y
		p.scene.absolute = append(p.scene.absolute, e)
		return nil
	}
	if node.Tag == "br" {
		p.write(owner, "\n")
		p.pending = false
		return nil
	}
	if e.css.Keyword("display") == "inline-block" || node.Tag == "img" {
		return p.appendObject(e)
	}
	if e.css.Keyword("display") == "block" {
		return e.reject("display") // Anonymous block splitting is handled separately.
	}
	return p.appendInline(e)
}

func (p *paragraph) appendText(source string, owner *element) error {
	mode := owner.css.Keyword("white-space")
	if mode == "pre" || mode == "pre-wrap" {
		// A fixed eight-space replacement is not a tab stop. Until layout owns
		// line-relative tab advances, require authors to supply explicit spaces.
		if strings.ContainsRune(source, '\t') {
			return owner.reject("white-space")
		}
		p.flushSpace(owner)
		p.write(owner, source)
		return nil
	}
	p.appendCollapsed(source, owner, mode == "pre-line")
	return nil
}

func (p *paragraph) appendCollapsed(source string, owner *element, preserveLines bool) {
	var text strings.Builder
	for _, r := range source {
		if cssSpace(r) {
			if preserveLines && r == '\n' {
				text.WriteRune('\n')
				p.pending = false
			} else {
				p.pending = true
			}
			continue
		}
		if p.pending && (p.text.Len() > 0 || text.Len() > 0) {
			text.WriteByte(' ')
		}
		p.pending = false
		text.WriteRune(r)
	}
	p.write(owner, text.String())
}

func (p *paragraph) flushSpace(owner *element) {
	if p.pending && p.text.Len() > 0 {
		p.write(owner, " ")
	}
	p.pending = false
}

func (p *paragraph) appendObject(e *element) error {
	p.flushSpace(e)
	area := geometry.Rect{Width: p.width, Height: p.root.box.ContentHeight}
	if err := p.scene.layoutBlock(e, p.scene.reference(area, p.root.box.DefiniteHeight), 0, 0); err != nil {
		return err
	}
	w, h := e.box.OuterWidth()+e.box.Margin.Horizontal(), e.box.OuterHeight()+e.box.Margin.Vertical()
	if w < 0 || h < 0 {
		return e.reject("margin")
	}
	object := canvas.New(w, h)
	p.objects[object] = e
	p.writeObject(e, object)
	return nil
}

func (p *paragraph) write(owner *element, source string) {
	if source == "" {
		return
	}
	p.runs = append(p.runs, inlineRun{owner: owner, source: source})
	p.text.WriteFace(p.face(owner), source)
}
