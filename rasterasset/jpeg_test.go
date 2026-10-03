package rasterasset

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestJPEGIsBoundedBeforeDecode(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 8, 8))
	src.Set(0, 0, color.White)
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, src, nil); err != nil {
		t.Fatal(err)
	}
	url := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes())
	got, err := DecodeJPEGDataURL(url, testLimits(2048, 8, 64))
	if err != nil || got.Bounds() != src.Bounds() {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := DecodeJPEGDataURL(url, testLimits(2048, 7, 64)); !errors.Is(err, ErrLimit) {
		t.Fatalf("dimension limit: %v", err)
	}
	if _, err := DecodeJPEGDataURL(url, testLimits(2048, 8, 63)); !errors.Is(err, ErrLimit) {
		t.Fatalf("pixel limit: %v", err)
	}
	if _, err := DecodeJPEGDataURL(dataURL(encoded.Bytes()), testLimits(2048, 8, 64)); !errors.Is(err, ErrSource) {
		t.Fatalf("wrong MIME: %v", err)
	}
}

func TestJPEGRejectsTruncationAndWrongImage(t *testing.T) {
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	data := encoded.Bytes()
	for _, invalid := range [][]byte{data[:10], data[:len(data)-1], encodePNG(t, image.NewGray(image.Rect(0, 0, 1, 1)))} {
		url := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(invalid)
		if _, err := DecodeJPEGDataURL(url, testLimits(2048, 10, 100)); !errors.Is(err, ErrImage) {
			t.Fatalf("invalid JPEG: %v", err)
		}
	}
}
