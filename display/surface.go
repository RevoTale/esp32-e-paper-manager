// Package display defines device-independent monochrome frame and lifecycle
// contracts. Frames borrow caller-owned storage and never allocate or copy it.
package display

import "errors"

var (
	ErrCapabilities  = errors.New("display: invalid capabilities")
	ErrColor         = errors.New("display: invalid color")
	ErrFrameGeometry = errors.New("display: invalid frame geometry")
	ErrPixelBounds   = errors.New("display: pixel outside frame")
	ErrRefreshMode   = errors.New("display: unsupported refresh mode")
)

// Size is a logical display size in pixels.
type Size struct {
	Width  int
	Height int
}

// Color is a monochrome pixel value. Packed frames use zero for white and one
// for black, with the leftmost pixel in the most-significant bit.
type Color uint8

const (
	White Color = iota
	Black
)

// ColorModel identifies a frame's pixel representation.
type ColorModel uint8

const (
	Mono1 ColorModel = 1 + iota
)

// RefreshMode is both a single refresh request and a capabilities bit.
type RefreshMode uint8

const (
	RefreshFull RefreshMode = 1 << iota
	RefreshPartial
)

// Capabilities describes the logical frame and lifecycle accepted by a Device.
type Capabilities struct {
	Size              Size
	ColorModel        ColorModel
	RefreshModes      RefreshMode
	WidthAlignment    int
	HeightAlignment   int
	StrideAlignment   int
	RefreshAutoSleeps bool
}

// Validate reports whether the capabilities form a usable monochrome contract.
func (c Capabilities) Validate() error {
	if !c.validGeometry() || !c.validModes() || c.ColorModel != Mono1 {
		return ErrCapabilities
	}
	return nil
}

func (c Capabilities) validGeometry() bool {
	return c.Size.Width > 0 && c.Size.Height > 0 &&
		c.WidthAlignment > 0 && c.HeightAlignment > 0 && c.StrideAlignment > 0 &&
		c.Size.Width%c.WidthAlignment == 0 && c.Size.Height%c.HeightAlignment == 0
}

func (c Capabilities) validModes() bool {
	return c.RefreshModes&RefreshFull != 0 &&
		c.RefreshModes&^(RefreshFull|RefreshPartial) == 0
}

// FrameStride returns the one packed row stride accepted by the device.
func (c Capabilities) FrameStride() (int, error) {
	if err := c.Validate(); err != nil {
		return 0, err
	}
	stride := packedStride(c.Size.Width)
	remainder := stride % c.StrideAlignment
	if remainder == 0 {
		return stride, nil
	}
	padding := c.StrideAlignment - remainder
	if stride > int(^uint(0)>>1)-padding {
		return 0, ErrCapabilities
	}
	return stride + padding, nil
}

// ValidateFrame verifies that frame and mode can be consumed by the device.
func (c Capabilities) ValidateFrame(frame Frame, mode RefreshMode) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if !singleRefreshMode(mode) || c.RefreshModes&mode == 0 {
		return ErrRefreshMode
	}
	stride, err := c.FrameStride()
	if err != nil {
		return err
	}
	if frame.size != c.Size || frame.stride != stride {
		return ErrFrameGeometry
	}
	return nil
}

func singleRefreshMode(mode RefreshMode) bool {
	return mode == RefreshFull || mode == RefreshPartial
}

// Frame is a mutable view over caller-owned packed monochrome storage. Copies
// of Frame share the same bytes. The storage must remain stable while Refresh
// is running and may be reused as soon as Refresh returns.
type Frame struct {
	size   Size
	stride int
	pixels []byte
}

// NewFrame validates and borrows pixels without copying them.
func NewFrame(size Size, stride int, pixels []byte) (Frame, error) {
	minimumStride := packedStride(size.Width)
	if size.Width <= 0 || size.Height <= 0 || stride < minimumStride ||
		stride > int(^uint(0)>>1)/size.Height || len(pixels) != stride*size.Height {
		return Frame{}, ErrFrameGeometry
	}
	return Frame{size: size, stride: stride, pixels: pixels}, nil
}

func packedStride(width int) int {
	stride := width / 8
	if width%8 != 0 {
		stride++
	}
	return stride
}

// Size returns the logical frame size.
func (f Frame) Size() Size { return f.size }

// Stride returns bytes between adjacent rows.
func (f Frame) Stride() int { return f.stride }

// Bytes returns the borrowed backing storage. Callers must follow Frame's
// ownership rules and must not resize the slice.
func (f Frame) Bytes() []byte { return f.pixels }

// Pixel returns the pixel at x, y. An out-of-bounds coordinate returns White.
func (f Frame) Pixel(x, y int) Color {
	if !f.contains(x, y) {
		return White
	}
	if f.pixels[y*f.stride+x/8]&(0x80>>uint(x&7)) != 0 {
		return Black
	}
	return White
}

// SetPixel changes one pixel without allocating.
func (f Frame) SetPixel(x, y int, color Color) error {
	if !validColor(color) {
		return ErrColor
	}
	if !f.contains(x, y) {
		return ErrPixelBounds
	}
	index := y*f.stride + x/8
	mask := byte(0x80 >> uint(x&7))
	if color == Black {
		f.pixels[index] |= mask
	} else {
		f.pixels[index] &^= mask
	}
	return nil
}

// Clear fills all logical and padding bits with color.
func (f Frame) Clear(color Color) error {
	if !validColor(color) {
		return ErrColor
	}
	value := byte(0)
	if color == Black {
		value = 0xff
	}
	for index := range f.pixels {
		f.pixels[index] = value
	}
	if color == Black && f.size.Width&7 != 0 {
		mask := byte(0xff << uint(8-f.size.Width&7))
		for y := 0; y < f.size.Height; y++ {
			f.pixels[y*f.stride+(f.size.Width-1)/8] &= mask
		}
	}
	return nil
}

func validColor(color Color) bool { return color == White || color == Black }

func (f Frame) contains(x, y int) bool {
	return x >= 0 && y >= 0 && x < f.size.Width && y < f.size.Height
}

// Device consumes complete frames synchronously. Refresh must not retain or
// mutate frame storage after it returns. Sleep is idempotent.
type Device interface {
	Capabilities() Capabilities
	Refresh(frame Frame, mode RefreshMode) error
	Sleep() error
}
