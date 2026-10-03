package engine

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// Preserve blitzworker/image_extra_test.go's normalization guard, not merely
// the rasterasset decoder's high-bit-depth/palette preservation contract.
func TestMigrationPaletteAnd16BitNormalizeToStraightRGBA(t *testing.T) {
	palette := image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.NRGBA{R: 255, A: 128}})
	wide := image.NewNRGBA64(image.Rect(0, 0, 1, 1))
	wide.SetNRGBA64(0, 0, color.NRGBA64{R: 65535, A: 32896})
	for _, source := range []image.Image{palette, wide} {
		var data bytes.Buffer
		if err := png.Encode(&data, source); err != nil {
			t.Fatal(err)
		}
		url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data.Bytes())
		img, err := decodeAsset(url, 1)
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds() != image.Rect(0, 0, 1, 1) || !bytes.Equal(img.Pix, []byte{255, 0, 0, 128}) {
			t.Fatal("normalization lost color/alpha or premultiplied it", img.Pix)
		}
	}
}
