package engine

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

type painter struct {
	scene      *scene
	target     *image.RGBA
	visits     int
	operations int
	temporary  int
	groups     int
}

func (p *painter) work(pixels int) error {
	if err := p.scene.ctx.Err(); err != nil {
		return err
	}
	if pixels < 0 || pixels > 64*1024*1024-p.visits || p.operations >= 65536 {
		return &renderdiag.Error{Code: renderdiag.InputLimit, Source: renderdiag.HTML}
	}
	p.visits += pixels
	p.operations++
	return nil
}

func (p *painter) allocate(bounds image.Rectangle) (*image.RGBA, error) {
	bytes := bounds.Dx() * bounds.Dy() * 4
	if bytes > 32*1024*1024-p.temporary {
		return nil, &renderdiag.Error{Code: renderdiag.InputLimit, Source: renderdiag.HTML}
	}
	p.temporary += bytes
	return image.NewRGBA(bounds), nil
}

func (p *painter) paintContext(e *element) error {
	if e.hidden {
		return nil
	}
	opacity := e.css.Number("opacity")
	if opacity >= 1 {
		return p.paintEntries(e, true)
	}
	if p.groups >= 8 {
		return &renderdiag.Error{Code: renderdiag.InputLimit, Source: renderdiag.HTML, Element: e.node.Ordinal}
	}
	original := p.target
	group, err := p.allocate(original.Bounds())
	if err != nil {
		return err
	}
	p.target, p.groups = group, p.groups+1
	defer func() { p.target, p.groups = original, p.groups-1; p.temporary -= len(group.Pix) }()
	if err := p.paintEntries(e, true); err != nil {
		return err
	}
	if err := p.work(group.Bounds().Dx() * group.Bounds().Dy()); err != nil {
		return err
	}
	mask := image.NewUniform(color.Alpha{A: uint8(math.Round(opacity * 255))})
	draw.DrawMask(original, original.Bounds(), group, image.Point{}, mask, image.Point{}, draw.Over)
	return nil
}

func (p *painter) paintEntries(e *element, includeContexts bool) error {
	if err := p.paintBox(e); err != nil {
		return err
	}
	for _, entry := range contextEntries(e, includeContexts) {
		if err := p.paintEntry(entry); err != nil {
			return err
		}
	}
	return nil
}

func (p *painter) paintEntry(entry paintEntry) error {
	if entry.context {
		return p.paintContext(entry.element)
	}
	if entry.atomic {
		return p.paintEntries(entry.element, false)
	}
	if entry.box {
		if err := p.paintBox(entry.element); err != nil {
			return err
		}
	}
	if entry.content {
		return p.paintContent(entry.element)
	}
	return nil
}
