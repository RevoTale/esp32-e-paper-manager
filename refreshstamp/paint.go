package refreshstamp

import (
	"errors"
	"image"

	"github.com/RevoTale/esp32-e-paper-manager/bitmapfont"
	"github.com/RevoTale/esp32-e-paper-manager/display"
)

var ErrOccupied = errors.New("refreshstamp: content occupies reserved corner")

// Bounds derives the reserved rectangle from the existing timestamp font and
// padding. A smaller display rejects instead of shrinking unreadable glyphs.
func Bounds(size display.Size) (image.Rectangle, error) {
	width, height := bitmapfont.TimestampDimensions()
	if size.Width < width || size.Height < height {
		return image.Rectangle{}, display.ErrFrameGeometry
	}
	return image.Rect(size.Width-width, size.Height-height, size.Width, size.Height), nil
}

// Paint modifies only an empty reserved rectangle in borrowed frame storage.
// Run after HTML composition and before diff/encoding. It never hides visible
// content; layout-level occupancy diagnostics must be enforced separately.
// Partial updates pass the exact label returned by Tracker.ForPartial.
func Paint(frame display.Frame, label Label) error {
	if !label.valid {
		return ErrTime
	}
	rect, err := Bounds(frame.Size())
	if err != nil {
		return err
	}
	if occupied(frame, rect) {
		return ErrOccupied
	}
	metrics := bitmapfont.ForSize(bitmapfont.TimestampSize)
	for y := 0; y < rect.Dy(); y++ {
		for x := 0; x < rect.Dx(); x++ {
			if stampInk(label, metrics, x, y, rect.Dx(), rect.Dy()) {
				px, py := rect.Min.X+x, rect.Min.Y+y
				frame.Bytes()[py*frame.Stride()+px/8] |= 0x80 >> uint(px%8)
			}
		}
	}
	return nil
}

func occupied(frame display.Frame, rect image.Rectangle) bool {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if frame.Pixel(x, y) != display.White {
				return true
			}
		}
	}
	return false
}

func stampInk(label Label, metrics bitmapfont.Metrics, x, y, width, height int) bool {
	if x == 0 || y == 0 || x == width-1 || y == height-1 {
		return true
	}
	x, y = x-bitmapfont.TimestampPad, y-bitmapfont.TimestampPad
	if x < 0 || y < 0 {
		return false
	}
	index := x / metrics.Advance
	return index < len(label.text) && bitmapfont.RunePixel(rune(label.text[index]), x%metrics.Advance, y, metrics)
}
