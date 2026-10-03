package engine

import (
	"context"
	"errors"
	"image"
	"image/draw"
	"strings"

	"github.com/RevoTale/esp32-e-paper-manager/engine/dom"
	"github.com/RevoTale/esp32-e-paper-manager/rasterasset"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

type assets struct {
	images, backgrounds map[*dom.Node]*image.NRGBA
	cache               map[string]*image.NRGBA
	count, pixels       int
}

func loadAssets(ctx context.Context, doc *dom.Document) (*assets, error) {
	a := &assets{images: make(map[*dom.Node]*image.NRGBA), backgrounds: make(map[*dom.Node]*image.NRGBA), cache: make(map[string]*image.NRGBA)}
	if err := a.visit(ctx, doc.Root); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *assets) visit(ctx context.Context, node *dom.Node) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if node.Tag == "img" {
		img, err := a.decode(node.Attrs["src"])
		if err != nil {
			return assetError(node, false, err)
		}
		a.images[node] = img
	}
	if background, ok := node.Style.Get("background-image"); ok && background.Image != "" {
		img, err := a.decode(background.Image)
		if err != nil {
			return assetError(node, true, err)
		}
		a.backgrounds[node] = img
	}
	for _, child := range node.Children {
		if err := a.visit(ctx, child); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func (a *assets) decode(source string) (*image.NRGBA, error) {
	a.count++
	if a.count > 32 || a.pixels >= 1_048_576 {
		return nil, rasterasset.ErrLimit
	}
	img := a.cache[source]
	if img == nil {
		var err error
		img, err = decodeAsset(source, 1_048_576-a.pixels)
		if err != nil {
			return nil, err
		}
		a.cache[source] = img
	}
	pixels := img.Bounds().Dx() * img.Bounds().Dy()
	if pixels > 1_048_576-a.pixels {
		return nil, rasterasset.ErrLimit
	}
	// Repeated uses share decoded memory but still consume scene work budget.
	a.pixels += pixels
	return img, nil
}

func decodeAsset(source string, pixels int) (*image.NRGBA, error) {
	limits := rasterasset.Limits{MaxURLBytes: 32768, MaxPNGBytes: 32768, MaxDimension: 2048, MaxPixels: pixels}
	decode := rasterasset.DecodeDataURL
	if strings.HasPrefix(source, "data:image/jpeg;") {
		decode = rasterasset.DecodeJPEGDataURL
	}
	decoded, err := decode(source, limits)
	if err != nil {
		return nil, err
	}
	// Normalize 16-bit/opaque decoder outputs to bounded four-byte straight RGBA.
	// Alpha is preserved; no monochrome/white-background shortcut is allowed here.
	result := image.NewNRGBA(decoded.Bounds())
	draw.Draw(result, result.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	return result, nil
}

func assetError(node *dom.Node, background bool, cause error) error {
	code := renderdiag.InvalidValue
	if errors.Is(cause, rasterasset.ErrLimit) {
		code = renderdiag.InputLimit
	}
	if background {
		return node.Style.Error("background-image", code)
	}
	return &renderdiag.Error{Code: code, Source: renderdiag.HTML, Element: node.Ordinal}
}
