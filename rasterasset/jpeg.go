package rasterasset

import (
	"bytes"
	"image"
	"image/jpeg"
)

// DecodeJPEGDataURL accepts canonical JPEG data URLs with the same allocation
// limits as PNG. JPEG is opaque and lossy; it is not compared pixel-for-pixel
// with its pre-encoding source. DecodeConfig precedes pixel allocation.
// https://pkg.go.dev/image#hdr-Security_Considerations
func DecodeJPEGDataURL(src string, limits Limits) (image.Image, error) {
	data, err := decodeSource(src, "data:image/jpeg;base64,", limits)
	if err != nil {
		return nil, err
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrImage
	}
	if !withinLimits(config, limits) {
		return nil, ErrLimit
	}
	// This profile requires an EOI at the end; malformed/truncated streams never
	// substitute an image. The standard decoder remains the format authority.
	if len(data) < 2 || data[len(data)-2] != 0xff || data[len(data)-1] != 0xd9 {
		return nil, ErrImage
	}
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrImage
	}
	return img, nil
}
