package rasterasset

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestDecodeDataURLPreservesPixelsAndAlpha(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	source.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 128})
	source.SetNRGBA(1, 0, color.NRGBA{B: 255})
	data := encodePNG(t, source)
	got, err := DecodeDataURL(dataURL(data), testLimits(len(data), 2, 2))
	if err != nil || got.Bounds() != source.Bounds() {
		t.Fatalf("image=%v error=%v", got, err)
	}
	for x := range 2 {
		if pixel := color.NRGBAModel.Convert(got.At(x, 0)); pixel != source.NRGBAAt(x, 0) {
			t.Fatalf("pixel %d: got %v, want %v", x, pixel, source.NRGBAAt(x, 0))
		}
	}
}

func TestDecodeDataURLRejectsSourcesAndNonCanonicalBase64(t *testing.T) {
	for _, src := range []string{
		"", "https://example.invalid/a.png", "file:///etc/passwd",
		"data:image/jpeg;base64,/9j/", "data:image/svg+xml;base64,PHN2Zz4=",
		"data:image/png,abc", "data:image/png;charset=utf-8;base64,AA==",
		"data:image/png;base64,", "data:image/png;base64,AQ",
		"data:image/png;base64,AR==", "data:image/png;base64,AA==\r\n",
		"data:image/png;base64,AA== ", "data:image/png;base64,AA%3D%3D",
	} {
		t.Run(src, func(t *testing.T) {
			img, err := DecodeDataURL(src, testLimits(1024, 100, 100))
			if img != nil || !errors.Is(err, ErrSource) {
				t.Fatalf("image=%v error=%v", img, err)
			}
		})
	}
}

func TestDecodeDataURLRejectsTruncatedAndCorruptPNG(t *testing.T) {
	data := encodePNG(t, image.NewGray(image.Rect(0, 0, 2, 2)))
	for length := 1; length < len(data); length++ {
		img, err := DecodeDataURL(dataURL(data[:length]), testLimits(1024, 100, 100))
		if img != nil || !errors.Is(err, ErrImage) {
			t.Fatalf("length=%d image=%v error=%v", length, img, err)
		}
	}
	data[len(data)-1] ^= 1 // IEND CRC must still be checked after decoding pixels.
	if _, err := DecodeDataURL(dataURL(data), testLimits(1024, 100, 100)); !errors.Is(err, ErrImage) {
		t.Fatalf("CRC: %v", err)
	}
}

func TestDecodeDataURLRejectsDimensionsBeforePixelDecode(t *testing.T) {
	data := encodePNG(t, image.NewGray(image.Rect(0, 0, 2, 2)))
	for _, size := range [][2]uint32{{2049, 1}, {1, 2049}, {2048, 1024}} {
		// Valid IHDR/CRC, but the tiny IDAT cannot possibly describe this image.
		// ErrLimit rather than ErrImage proves rejection before pixel decoding.
		binary.BigEndian.PutUint32(data[16:20], size[0])
		binary.BigEndian.PutUint32(data[20:24], size[1])
		binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
		img, err := DecodeDataURL(dataURL(data), testLimits(1024, 2048, 1_048_576))
		if img != nil || !errors.Is(err, ErrLimit) {
			t.Fatalf("size=%v image=%v error=%v", size, img, err)
		}
	}
}

func TestDecodeDataURLRejectsEncodedAndSourceLimits(t *testing.T) {
	data := encodePNG(t, image.NewGray(image.Rect(0, 0, 2, 2)))
	for _, limits := range []Limits{
		{MaxURLBytes: len(dataURL(data)) - 1, MaxPNGBytes: 1024, MaxDimension: 10, MaxPixels: 100},
		testLimits(len(data)-1, 10, 100),
	} {
		img, err := DecodeDataURL(dataURL(data), limits)
		if img != nil || !errors.Is(err, ErrLimit) {
			t.Fatalf("image=%v error=%v", img, err)
		}
	}
	src := strings.Repeat("x", 4097)
	allocs := testing.AllocsPerRun(10, func() {
		_, err := DecodeDataURL(src, testLimits(1024, 10, 100))
		if !errors.Is(err, ErrLimit) {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("oversized URL allocated %.0f times", allocs)
	}
}

func TestDecodeDataURLRejectsExcessPaddingWithoutAllocating(t *testing.T) {
	src := "data:image/png;base64," + strings.Repeat("=", 1024)
	allocs := testing.AllocsPerRun(10, func() {
		_, err := DecodeDataURL(src, testLimits(1, 1, 1))
		if !errors.Is(err, ErrSource) {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("malformed padding bypassed source bound: %.0f allocations", allocs)
	}
}

func TestDecodeDataURLRejectsInvalidLimits(t *testing.T) {
	valid := testLimits(100, 10, 100)
	for _, limits := range []Limits{
		{}, {MaxURLBytes: -1, MaxPNGBytes: 100, MaxDimension: 10, MaxPixels: 100},
		{MaxURLBytes: valid.MaxURLBytes, MaxDimension: 10, MaxPixels: 100},
		{MaxURLBytes: valid.MaxURLBytes, MaxPNGBytes: 100, MaxPixels: 100},
		{MaxURLBytes: valid.MaxURLBytes, MaxPNGBytes: 100, MaxDimension: 10},
	} {
		if _, err := DecodeDataURL("", limits); !errors.Is(err, ErrConfiguration) {
			t.Fatalf("limits=%+v error=%v", limits, err)
		}
	}
}

func encodePNG(t testing.TB, source image.Image) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, source); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func dataURL(data []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}

func testLimits(maxPNGBytes, maxDimension, maxPixels int) Limits {
	return Limits{MaxURLBytes: 4096, MaxPNGBytes: maxPNGBytes, MaxDimension: maxDimension, MaxPixels: maxPixels}
}
