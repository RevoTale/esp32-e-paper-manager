package engine

import (
	"image"

	"github.com/RevoTale/esp32-e-paper-manager/engine/geometry"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

func (p *painter) drawImage(img *image.NRGBA, area geometry.Rect, clips []clip) error {
	bounds := pixelBounds(area, clips).Intersect(p.target.Bounds())
	if area.Empty() || bounds.Empty() {
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
	mapping := f64.Aff3{area.Width / float64(img.Bounds().Dx()), 0, area.X - float64(bounds.Min.X), 0, area.Height / float64(img.Bounds().Dy()), area.Y - float64(bounds.Min.Y)}
	// Bilinear CPU sampling works on premultiplied colors, retaining source alpha
	// until it is composited over the actual backdrop (never over implicit white).
	// https://pkg.go.dev/golang.org/x/image/draw#Interpolator
	draw.ApproxBiLinear.Transform(tile, mapping, img, img.Bounds(), draw.Src, nil)
	return p.composite(tile, bounds, clips)
}

func (p *painter) paintImage(e *element, clips []clip) error {
	img := p.scene.assets.images[e.node]
	if img == nil {
		return nil
	}
	content := e.contentRect()
	area, err := geometry.ObjectRect(content, float64(img.Bounds().Dx()), float64(img.Bounds().Dy()), e.css, p.scene.reference(content, true))
	if err != nil {
		return e.reject("object-fit")
	}
	b, padding := e.box.Border, e.box.Padding
	inner, err := p.innerClip(e, geometry.Edges{Top: b.Top + padding.Top, Right: b.Right + padding.Right, Bottom: b.Bottom + padding.Bottom, Left: b.Left + padding.Left})
	if err != nil {
		return err
	}
	return p.drawImage(img, area, append(clips, inner))
}

func (p *painter) paintBackground(e *element, area geometry.Rect, radius, maxHeight float64, clips []clip) error {
	img := p.scene.assets.backgrounds[e.node]
	if img == nil || area.Empty() {
		return nil
	}
	// Default background-origin is padding-box; default background-clip remains
	// border-box, so repeated tiles can still appear behind transparent borders.
	// https://www.w3.org/TR/css-backgrounds-3/#the-background-origin
	origin := insetRect(area, e.box.Border)
	tile, err := p.backgroundRect(e, origin, maxHeight-e.box.Border.Vertical(), float64(img.Bounds().Dx()), float64(img.Bounds().Dy()))
	if err != nil {
		return e.reject("background-size-x")
	}
	visible := area.Intersect(p.scene.viewport)
	for _, c := range clips {
		visible = visible.Intersect(c.area)
	}
	tiles, err := p.backgroundTiles(e, tile, visible)
	if err != nil {
		return err
	}
	defer func() { p.temporary -= cap(tiles) * 32 }()
	for _, target := range tiles {
		if err := p.drawImage(img, target, append(clips, clip{area: area, radius: radius})); err != nil {
			return err
		}
	}
	return nil
}

func (p *painter) backgroundTiles(e *element, tile, visible geometry.Rect) ([]geometry.Rect, error) {
	// Four float64 coordinates per entry. Account for retained capacity before
	// allowing any per-tile raster allocation, not after the slice is discarded.
	budget := min(65536-p.operations, (32*1024*1024-p.temporary)/32)
	if budget <= 0 {
		return nil, e.node.Style.Error("background-repeat", renderdiag.InputLimit)
	}
	tiles, err := geometry.Tiles(tile, visible, e.css.Keyword("background-repeat"), budget)
	if err != nil {
		return nil, e.node.Style.Error("background-repeat", renderdiag.InputLimit)
	}
	p.temporary += cap(tiles) * 32
	return tiles, nil
}
