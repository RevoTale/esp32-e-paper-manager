package engine

import "github.com/tdewolff/canvas"

func (p *paragraph) writeObject(owner *element, object *canvas.Canvas) {
	// Pinned Canvas dae8cd8e19a7 RichText.ToText can merge text after a forced
	// line break into a same-face object span; RenderTextTo then omits its glyphs.
	// A distinct object face preserves the existing span boundary. Metrics and
	// font/shaper remain shared; only one small descriptor per object is copied.
	// https://github.com/tdewolff/canvas/blob/dae8cd8e19a7/text.go
	face := *p.face(owner)
	p.owners[&face] = owner
	p.runs = append(p.runs, inlineRun{owner: owner, object: object, face: &face})
	p.text.SetFace(&face)
	p.text.WriteCanvas(object, canvas.Baseline)
}
