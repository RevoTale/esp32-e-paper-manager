// Package rasterasset validates manager-side image input without file/network
// access, HTML parsing, panel dependencies or early monochrome composition.
package rasterasset

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/png"
	"strings"
)

var (
	ErrConfiguration = errors.New("rasterasset: invalid limits")
	ErrSource        = errors.New("rasterasset: expected canonical raster data URL")
	ErrLimit         = errors.New("rasterasset: image exceeds limit")
	ErrImage         = errors.New("rasterasset: invalid raster image")
)

// Limits are positive administrator-supplied per-image bounds, not HTML values.
// MaxURLBytes includes the prefix/base64; MaxPNGBytes bounds the decoded source
// file, not its pixels (the legacy field name also applies to JPEG). Callers
// must also enforce a document-wide asset budget.
type Limits struct {
	MaxURLBytes, MaxPNGBytes, MaxDimension, MaxPixels int
}

// DecodeDataURL accepts only data:image/png;base64, with padded RFC4648 standard
// base64 and no whitespace. It returns caller-owned pixels with alpha intact;
// the renderer must composite against the actual scene before thresholding.
// No URL resolver, global image decoder registry, or source metadata is exposed.
func DecodeDataURL(src string, limits Limits) (image.Image, error) {
	data, err := decodeSource(src, "data:image/png;base64,", limits)
	if err != nil {
		return nil, err
	}
	return decodePNG(data, limits)
}

func decodeSource(src, prefix string, limits Limits) ([]byte, error) {
	if min(limits.MaxURLBytes, limits.MaxPNGBytes, limits.MaxDimension, limits.MaxPixels) <= 0 {
		return nil, ErrConfiguration
	}
	if len(src) > limits.MaxURLBytes {
		return nil, ErrLimit
	}
	payload, ok := strings.CutPrefix(src, prefix)
	if !ok || len(payload) == 0 || len(payload)%4 != 0 || strings.ContainsAny(payload, " \t\r\n") {
		return nil, ErrSource
	}
	size := base64.StdEncoding.DecodedLen(len(payload))
	padding := len(payload) - len(strings.TrimRight(payload, "="))
	if padding > 2 {
		return nil, ErrSource
	}
	size -= padding
	if size > limits.MaxPNGBytes {
		return nil, ErrLimit
	}
	// Strict rejects nonzero trailing pad bits, but still permits CR/LF; the
	// explicit whitespace check above closes that gap in our canonical profile.
	// https://pkg.go.dev/encoding/base64#Encoding.Strict
	data, err := base64.StdEncoding.Strict().DecodeString(payload)
	if err != nil {
		return nil, ErrSource
	}
	return data, nil
}

func decodePNG(data []byte, limits Limits) (image.Image, error) {
	// DecodeConfig before Decode is Go's documented untrusted-image guard:
	// https://pkg.go.dev/image#hdr-Security_Considerations
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrImage
	}
	if !withinLimits(config, limits) {
		return nil, ErrLimit
	}
	reader := bytes.NewReader(data)
	img, err := png.Decode(reader)
	if err != nil || reader.Len() != 0 {
		return nil, ErrImage
	}
	return img, nil
}

func withinLimits(config image.Config, limits Limits) bool {
	return config.Width > 0 && config.Height > 0 && config.Width <= limits.MaxDimension &&
		config.Height <= limits.MaxDimension && config.Width <= limits.MaxPixels/config.Height
}
