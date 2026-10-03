package engine

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

func (p *painter) drawPath(path *canvas.Path, style canvas.Style, world canvas.Matrix, clips []clip) error {
	b := path.Copy().Transform(world).FastBounds()
	area := geometry.Rect{X: b.X0, Y: b.Y0, Width: b.W(), Height: b.H()}
	bounds := pixelBounds(area, clips).Intersect(p.target.Bounds())
	if bounds.Empty() {
		return p.work(0)
	}
	if err := p.work(bounds.Dx() * bounds.Dy() * 2); err != nil {
		return err
	}
	tile, err := p.allocate(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	if err != nil {
		return err
	}
	defer func() { p.temporary -= len(tile.Pix) }()
	ras := rasterizer.FromImage(tile, canvas.DPMM(1), canvas.LinearColorSpace{})
	view := canvas.Identity.Translate(-float64(bounds.Min.X), float64(bounds.Max.Y)).Scale(1, -1).Mul(world)
	ras.RenderPath(path, style, view)
	return p.composite(tile, bounds, clips)
}

func (p *painter) composite(tile *image.RGBA, bounds image.Rectangle, clips []clip) error {
	// Rectangle/rounded ancestor clipping applies to positioned descendants too,
	// independently of where their stacking-context entry is scheduled.
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			if !insideClips(clips, float64(bounds.Min.X+x)+.5, float64(bounds.Min.Y+y)+.5) {
				tile.SetRGBA(x, y, color.RGBA{})
			}
		}
	}
	draw.Draw(p.target, bounds, tile, image.Point{}, draw.Over)
	return p.scene.ctx.Err()
}

func (p *painter) fillRect(area geometry.Rect, radius float64, fill color.NRGBA, clips []clip) error {
	if area.Empty() || fill.A == 0 {
		return nil
	}
	style := canvas.DefaultStyle
	style.Fill = canvas.Paint{Color: color.RGBAModel.Convert(fill).(color.RGBA)}
	path := canvas.RoundedRectangle(area.Width, area.Height, radius)
	return p.drawPath(path, style, canvas.Identity.Translate(area.X, area.Y), clips)
}
