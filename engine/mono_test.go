package engine

import (
	"context"
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func TestMonoUsesCanonicalOddWidthStrideAndPolarity(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	frame, err := r.Render(context.Background(), display.Size{Width: 17, Height: 9}, []byte(`<div style="width:9px;height:9px;background:black"></div>`))
	if err != nil || frame.Stride() != 3 || len(frame.Bytes()) != 27 {
		t.Fatalf("frame %+v %v", frame, err)
	}
	for y := 0; y < 9; y++ {
		row := frame.Bytes()[y*3 : y*3+3]
		if row[0] != 255 || row[1] != 128 || row[2] != 0 {
			t.Fatalf("row %d = %v", y, row)
		}
	}
}

func TestMonoRejectsUnfinishedAndInvalidSurfaces(t *testing.T) {
	for _, img := range []*image.RGBA{nil, image.NewRGBA(image.Rectangle{}), image.NewRGBA(image.Rect(0, 0, 2049, 1))} {
		if _, err := packMono(context.Background(), img); !errors.Is(err, display.ErrFrameGeometry) {
			t.Fatalf("geometry: %v", err)
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	if _, err := packMono(context.Background(), img); !errors.Is(err, display.ErrColor) {
		t.Fatalf("uncomposited: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := packMono(ctx, img); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Render(ctx, display.Size{Width: 1, Height: 1}, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("render cancel: %v", err)
	}
}

func TestMonoDitherIsDisplayAnchoredAndSourceImmutable(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	frame, err := packMono(context.Background(), img)
	if err != nil {
		t.Fatal(err)
	}
	black := 0
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if frame.Pixel(x, y) == display.Black {
				black++
			}
			if frame.Pixel(x, y) != frame.Pixel(x%4, y%4) || img.RGBAAt(x, y).R != 128 {
				t.Fatal("dither depends on scan history or changed source")
			}
		}
	}
	if black != 32 {
		t.Fatalf("mid-gray black pixels %d, want 32", black)
	}
}
