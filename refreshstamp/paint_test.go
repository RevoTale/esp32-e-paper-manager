package refreshstamp

import (
	"bytes"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func testFrame(t *testing.T, width, height, padding int) display.Frame {
	t.Helper()
	stride := (width+7)/8 + padding
	frame, err := display.NewFrame(display.Size{Width: width, Height: height}, stride, make([]byte, stride*height))
	if err != nil {
		t.Fatal(err)
	}
	return frame
}

func testLabel(t *testing.T) Label {
	t.Helper()
	label, err := testTracker(t).Begin(1, time.Date(2026, 9, 6, 12, 34, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return label
}

func TestPaintOwnsOnlyMetricDerivedCorner(t *testing.T) {
	label := testLabel(t)
	for _, size := range []display.Size{{Width: 800, Height: 480}, {Width: 129, Height: 33}} {
		frame := testFrame(t, size.Width, size.Height, 2)
		rect, err := Bounds(size)
		if err != nil || rect != image.Rect(size.Width-120, size.Height-21, size.Width, size.Height) {
			t.Fatalf("bounds = %v, %v", rect, err)
		}
		seedOutside(frame, rect)
		before := bytes.Clone(frame.Bytes())
		if err := Paint(frame, label); err != nil {
			t.Fatal(err)
		}
		assertOutsideUnchanged(t, frame, before, rect)
		if frame.Pixel(rect.Min.X, rect.Min.Y) != display.Black || frame.Pixel(rect.Max.X-1, rect.Max.Y-1) != display.Black {
			t.Fatal("missing timestamp border")
		}
		if frame.Pixel(rect.Min.X+2, rect.Min.Y+2) != display.White {
			t.Fatal("padding not white")
		}
	}
}

func seedOutside(frame display.Frame, rect image.Rectangle) {
	for i := range frame.Bytes() {
		frame.Bytes()[i] = 0xa5
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			frame.Bytes()[y*frame.Stride()+x/8] &^= 0x80 >> uint(x%8)
		}
	}
}

func assertOutsideUnchanged(t *testing.T, frame display.Frame, before []byte, rect image.Rectangle) {
	t.Helper()
	for y := 0; y < frame.Size().Height; y++ {
		for x := 0; x < frame.Stride()*8; x++ {
			if image.Pt(x, y).In(rect) {
				continue
			}
			i, mask := y*frame.Stride()+x/8, byte(0x80>>uint(x%8))
			if before[i]&mask != frame.Bytes()[i]&mask {
				t.Fatalf("changed non-corner bit at %d,%d", x, y)
			}
		}
	}
}

func TestPaintRejectsBeforeMutation(t *testing.T) {
	label := testLabel(t)
	for _, size := range []display.Size{{}, {Width: 119, Height: 21}, {Width: 120, Height: 20}} {
		if _, err := Bounds(size); !errors.Is(err, display.ErrFrameGeometry) {
			t.Fatalf("small geometry accepted: %v", err)
		}
	}
	if err := Paint(display.Frame{}, label); !errors.Is(err, display.ErrFrameGeometry) {
		t.Fatalf("zero frame accepted: %v", err)
	}
	frame := testFrame(t, 128, 32, 0)
	before := bytes.Clone(frame.Bytes())
	if err := Paint(frame, Label{}); !errors.Is(err, ErrTime) || !bytes.Equal(frame.Bytes(), before) {
		t.Fatalf("invalid label mutated frame: %v", err)
	}
	frame.Bytes()[len(frame.Bytes())-1] = 1
	before = bytes.Clone(frame.Bytes())
	if err := Paint(frame, label); !errors.Is(err, ErrOccupied) || !bytes.Equal(frame.Bytes(), before) {
		t.Fatalf("occupied corner was overwritten: %v", err)
	}
}
