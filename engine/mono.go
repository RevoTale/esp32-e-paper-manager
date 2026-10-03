package engine

import (
	"context"
	"image"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

// Render satisfies the manager's ScreenRenderer contract without depending on
// manager, transports or a particular panel. Logical size precedes rasterization.
func (r *Renderer) Render(ctx context.Context, size display.Size, html []byte) (display.Frame, error) {
	frame, _, err := r.RenderDetailed(ctx, size, html)
	return frame, err
}

// RenderDetailed adds source-free warnings without coupling the engine to a
// manager implementation. Returned warnings and frame belong to the caller.
func (r *Renderer) RenderDetailed(ctx context.Context, size display.Size, html []byte) (display.Frame, []renderdiag.Warning, error) {
	return r.RenderReserved(ctx, size, html, image.Rectangle{})
}

// RenderReserved clears only the protected metadata corner before quantization
// and preserves composition warnings. The manager paints its timestamp later.
func (r *Renderer) RenderReserved(ctx context.Context, size display.Size, html []byte, area image.Rectangle) (display.Frame, []renderdiag.Warning, error) {
	result, err := r.RenderRGBAWithOptions(ctx, size, html, Options{Reserved: area})
	if err != nil {
		return display.Frame{}, nil, err
	}
	frame, err := result.Frame(ctx)
	return frame, result.Warnings, err
}

// Frame quantizes the completed preview into independently owned canonical
// packed pixels. It never scales, rotates or mutates the preview.
func (r Result) Frame(ctx context.Context) (display.Frame, error) {
	return packMono(ctx, r.Image)
}

// Ordered dithering is anchored at logical display (0,0). Unlike error diffusion,
// an edit cannot propagate quantization error across otherwise unchanged rows.
var bayer4 = [4][4]uint32{{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5}}

func packMono(ctx context.Context, img *image.RGBA) (display.Frame, error) {
	if img == nil {
		return display.Frame{}, display.ErrFrameGeometry
	}
	bounds := img.Bounds()
	size := display.Size{Width: bounds.Dx(), Height: bounds.Dy()}
	if !validSize(size) {
		return display.Frame{}, display.ErrFrameGeometry
	}
	stride := (size.Width + 7) / 8
	pixels := make([]byte, stride*size.Height)
	for y := 0; y < size.Height; y++ {
		if err := ctx.Err(); err != nil {
			return display.Frame{}, err
		}
		for x := 0; x < size.Width; x++ {
			c := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			if c.A != 255 {
				return display.Frame{}, display.ErrColor // Composition must finish first.
			}
			luma := (19595*uint32(c.R) + 38470*uint32(c.G) + 7471*uint32(c.B) + 32768) >> 16
			if luma < bayer4[y&3][x&3]*16+8 {
				pixels[y*stride+x/8] |= 0x80 >> uint(x&7)
			}
		}
	}
	return display.NewFrame(size, stride, pixels)
}
