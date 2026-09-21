package rasterasset

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"testing"
)

func TestDecodeDataURLRejectsTrailingBytesAndConcatenatedImages(t *testing.T) {
	data := encodePNG(t, image.NewGray(image.Rect(0, 0, 1, 1)))
	for _, tail := range [][]byte{{0}, []byte("private metadata"), data} {
		src := append(bytes.Clone(data), tail...)
		if _, err := DecodeDataURL(dataURL(src), testLimits(1024, 10, 100)); !errors.Is(err, ErrImage) {
			t.Fatalf("trailing bytes: %v", err)
		}
	}
}

func TestDecodeDataURLPreservesSixteenBitAlphaAndPalette(t *testing.T) {
	gray := image.NewGray16(image.Rect(0, 0, 2, 1))
	gray.SetGray16(0, 0, color.Gray16{Y: 0x1234})
	alpha := image.NewNRGBA64(image.Rect(0, 0, 2, 1))
	alpha.SetNRGBA64(0, 0, color.NRGBA64{R: 65535, G: 16000, B: 32000, A: 12345})
	palette := image.NewPaletted(image.Rect(0, 0, 2, 1), color.Palette{
		color.NRGBA{R: 255, A: 128}, color.NRGBA{B: 255, A: 255},
	})
	palette.SetColorIndex(1, 0, 1)
	for _, source := range []image.Image{gray, alpha, palette} {
		data := encodePNG(t, source)
		got, err := DecodeDataURL(dataURL(data), testLimits(len(data), 2, 2))
		if err != nil {
			t.Fatal(err)
		}
		for x := range 2 {
			if color.NRGBA64Model.Convert(got.At(x, 0)) != color.NRGBA64Model.Convert(source.At(x, 0)) {
				t.Fatalf("%T: changed pixel at %d", source, x)
			}
		}
	}
}

func TestDecodeDataURLValidAtAllThreeBase64PaddingBoundaries(t *testing.T) {
	seen := [3]bool{}
	for width := 1; width <= 32; width++ {
		data := encodePNG(t, image.NewGray(image.Rect(0, 0, width, 1)))
		limits := testLimits(len(data), width, width)
		limits.MaxURLBytes = len(dataURL(data))
		if _, err := DecodeDataURL(dataURL(data), limits); err != nil {
			t.Fatalf("width=%d length=%d error=%v", width, len(data), err)
		}
		seen[len(data)%3] = true
	}
	if seen != [3]bool{true, true, true} {
		t.Fatalf("fixtures did not cover all padding lengths: %v", seen)
	}
}

func FuzzDecodeDataURL(f *testing.F) {
	f.Add("data:image/png;base64,AAAA")
	f.Add(dataURL(encodePNG(f, image.NewGray(image.Rect(0, 0, 2, 2)))))
	f.Fuzz(func(t *testing.T, src string) {
		limits := testLimits(1024, 32, 256)
		img, err := DecodeDataURL(src, limits)
		if err != nil {
			if img != nil {
				t.Fatal("failure returned pixels")
			}
			return
		}
		b := img.Bounds()
		if b.Dx() <= 0 || b.Dy() <= 0 || b.Dx() > 32 || b.Dy() > 32 || b.Dx()*b.Dy() > 256 {
			t.Fatalf("accepted invalid geometry: %v", b)
		}
	})
}

func FuzzPNGBytes(f *testing.F) {
	f.Add(encodePNG(f, image.NewGray(image.Rect(0, 0, 2, 2))))
	f.Add([]byte("not PNG"))
	f.Fuzz(func(t *testing.T, data []byte) {
		// Bound the fuzz harness's base64 allocation as well as the decoder.
		if len(data) > 1024 {
			return
		}
		img, err := DecodeDataURL(dataURL(data), testLimits(1024, 32, 256))
		if err != nil {
			if img != nil {
				t.Fatal("failure returned pixels")
			}
			return
		}
		b := img.Bounds()
		if b.Dx() <= 0 || b.Dy() <= 0 || b.Dx() > 32 || b.Dy() > 32 || b.Dx()*b.Dy() > 256 {
			t.Fatalf("accepted invalid geometry: %v", b)
		}
	})
}
