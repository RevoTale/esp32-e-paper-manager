package engine

import (
	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
	"github.com/tdewolff/canvas"
)

func (p *painter) paintBox(e *element) error {
	clips, err := p.clips(e)
	if err != nil {
		return err
	}
	if e.css.Keyword("display") == "inline" && e.node.Tag != "img" {
		return p.paintInlineBox(e, clips)
	}
	return p.paintDecoration(e, e.borderRect(), e.box.OuterHeight(), clips)
}

func (p *painter) paintDecoration(e *element, area geometry.Rect, maxHeight float64, clips []clip) error {
	radius, err := p.radius(e, area)
	if err != nil {
		return err
	}
	if err := p.fillRect(area, radius, e.css.Color("background-color"), clips); err != nil {
		return err
	}
	if err := p.paintBackground(e, area, radius, maxHeight, clips); err != nil {
		return err
	}
	return p.borderRing(e, area, radius, append(clips, clip{area: area, radius: radius}))
}

func (p *painter) paintContent(e *element) error {
	clips, err := p.clips(e)
	if err != nil {
		return err
	}
	if clipsOverflow(e) {
		own, err := p.innerClip(e, e.box.Border)
		if err != nil {
			return err
		}
		clips = append(clips, own)
	}
	for _, glyph := range e.glyphs {
		world := canvas.Identity.Translate(glyph.x, glyph.y).Scale(1, -1).Mul(glyph.view)
		if err := p.drawPath(glyph.path, glyph.style, world, clips); err != nil {
			return err
		}
	}
	return p.paintImage(e, clips)
}
