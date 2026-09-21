package engine

import (
	"context"
	"image"
	"image/color"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

type Options struct {
	// Reserved is a white, protected metadata rectangle in logical pixels.
	// The zero rectangle disables reservation; it does not reduce the viewport.
	Reserved image.Rectangle
}

func (o Options) validate(size display.Size) error {
	if !validSize(size) {
		return display.ErrFrameGeometry
	}
	a := o.Reserved
	if a == (image.Rectangle{}) {
		return nil
	}
	if a.Empty() || !a.In(image.Rect(0, 0, size.Width, size.Height)) {
		return display.ErrFrameGeometry
	}
	return nil
}

func (r *Result) reserve(ctx context.Context, area image.Rectangle) error {
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	occupied := false
	for y := area.Min.Y; y < area.Max.Y; y++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for x := area.Min.X; x < area.Max.X; x++ {
			occupied = occupied || r.Image.RGBAAt(x, y) != white
			r.Image.SetRGBA(x, y, white)
		}
	}
	if occupied {
		// Element zero identifies a composition-level warning, not an arbitrary
		// DOM owner: several transparent layers can contribute to this corner.
		r.Warnings = append(r.Warnings, renderdiag.Warning{Code: renderdiag.ReservedOverlap})
	}
	return nil
}
