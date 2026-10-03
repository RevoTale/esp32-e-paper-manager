// Package engine implements the bounded manager-side HTML rendering profile.
// It never imports MCU, panel, transport or credential packages.
package engine

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

type surface struct {
	image   *image.RGBA
	context *canvas.Context
}

func newSurface(size display.Size) (*surface, error) {
	if !validSize(size) {
		return nil, display.ErrFrameGeometry
	}
	img := image.NewRGBA(image.Rect(0, 0, size.Width, size.Height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	// One internal Canvas unit is one logical pixel, explicitly at 1 px/unit.
	// Canvas normally calls its units mm; never rely on its default 96 DPI.
	// https://github.com/tdewolff/canvas/blob/dae8cd8e19a7/renderers/rasterizer/rasterizer.go
	ras := rasterizer.FromImage(img, canvas.DPMM(1), canvas.LinearColorSpace{})
	ctx := canvas.NewContext(ras)
	ctx.SetCoordSystem(canvas.CartesianIV)
	return &surface{image: img, context: ctx}, nil
}

func validSize(size display.Size) bool {
	return size.Width > 0 && size.Height > 0 && size.Width <= 2048 && size.Height <= 2048 && size.Width <= 1_048_576/size.Height
}
