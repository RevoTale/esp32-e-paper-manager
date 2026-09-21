package display

import (
	"errors"
	"testing"
)

func TestNewFrameUsesCallerStorage(t *testing.T) {
	pixels := make([]byte, 4)
	frame, err := NewFrame(Size{Width: 13, Height: 2}, 2, pixels)
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}

	if err := frame.SetPixel(12, 1, Black); err != nil {
		t.Fatalf("SetPixel() error = %v", err)
	}
	if pixels[3] != 0x08 {
		t.Fatalf("caller storage byte = %#02x, want 0x08", pixels[3])
	}
	if got := frame.Pixel(12, 1); got != Black {
		t.Fatalf("Pixel() = %v, want Black", got)
	}
}

func TestNewFrameRejectsInvalidGeometry(t *testing.T) {
	tests := []struct {
		name   string
		size   Size
		stride int
		bytes  int
	}{
		{name: "zero width", size: Size{Height: 1}, stride: 1, bytes: 1},
		{name: "short stride", size: Size{Width: 9, Height: 1}, stride: 1, bytes: 1},
		{name: "short buffer", size: Size{Width: 8, Height: 2}, stride: 1, bytes: 1},
		{name: "trailing buffer", size: Size{Width: 8, Height: 1}, stride: 1, bytes: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewFrame(test.size, test.stride, make([]byte, test.bytes))
			if !errors.Is(err, ErrFrameGeometry) {
				t.Fatalf("NewFrame() error = %v, want ErrFrameGeometry", err)
			}
		})
	}
}

func TestFrameBoundsDoNotMutate(t *testing.T) {
	pixels := []byte{0xaa}
	frame, err := NewFrame(Size{Width: 8, Height: 1}, 1, pixels)
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}

	if err := frame.SetPixel(8, 0, Black); !errors.Is(err, ErrPixelBounds) {
		t.Fatalf("SetPixel() error = %v, want ErrPixelBounds", err)
	}
	if pixels[0] != 0xaa {
		t.Fatalf("out-of-bounds write changed frame to %#02x", pixels[0])
	}
}

func TestCapabilitiesValidateFrame(t *testing.T) {
	capabilities := Capabilities{
		Size:              Size{Width: 128, Height: 64},
		ColorModel:        Mono1,
		RefreshModes:      RefreshFull,
		WidthAlignment:    8,
		HeightAlignment:   1,
		StrideAlignment:   1,
		RefreshAutoSleeps: true,
	}
	pixels := make([]byte, 128*64/8)
	frame, err := NewFrame(capabilities.Size, 128/8, pixels)
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}
	if err := capabilities.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := capabilities.ValidateFrame(frame, RefreshFull); err != nil {
		t.Fatalf("ValidateFrame() error = %v", err)
	}
	if err := capabilities.ValidateFrame(frame, RefreshPartial); !errors.Is(err, ErrRefreshMode) {
		t.Fatalf("ValidateFrame(partial) error = %v, want ErrRefreshMode", err)
	}
	if got := frame.Size(); got != capabilities.Size {
		t.Fatalf("Size() = %+v, want %+v", got, capabilities.Size)
	}
	if got := frame.Stride(); got != 16 {
		t.Fatalf("Stride() = %d, want 16", got)
	}
}

func TestCapabilitiesRejectInvalidAndMismatchedFrames(t *testing.T) {
	invalid := Capabilities{}
	if err := invalid.Validate(); !errors.Is(err, ErrCapabilities) {
		t.Fatalf("Validate() error = %v, want ErrCapabilities", err)
	}
	frame, err := NewFrame(Size{Width: 16, Height: 8}, 2, make([]byte, 16))
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}
	if err := invalid.ValidateFrame(frame, RefreshFull); !errors.Is(err, ErrCapabilities) {
		t.Fatalf("ValidateFrame(invalid capabilities) error = %v", err)
	}
	capabilities := Capabilities{
		Size:            Size{Width: 32, Height: 8},
		ColorModel:      Mono1,
		RefreshModes:    RefreshFull,
		WidthAlignment:  8,
		HeightAlignment: 1,
		StrideAlignment: 2,
	}
	if err := capabilities.ValidateFrame(frame, RefreshFull); !errors.Is(err, ErrFrameGeometry) {
		t.Fatalf("ValidateFrame(mismatch) error = %v, want ErrFrameGeometry", err)
	}
}

func TestCapabilitiesRequireExactAlignedStride(t *testing.T) {
	capabilities := Capabilities{
		Size:            Size{Width: 17, Height: 2},
		ColorModel:      Mono1,
		RefreshModes:    RefreshFull,
		WidthAlignment:  1,
		HeightAlignment: 1,
		StrideAlignment: 4,
	}
	stride, err := capabilities.FrameStride()
	if err != nil {
		t.Fatalf("FrameStride() error = %v", err)
	}
	if stride != 4 {
		t.Fatalf("FrameStride() = %d, want 4", stride)
	}
	frame, err := NewFrame(capabilities.Size, 8, make([]byte, 16))
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}
	if err := capabilities.ValidateFrame(frame, RefreshFull); !errors.Is(err, ErrFrameGeometry) {
		t.Fatalf("ValidateFrame(padded) error = %v, want ErrFrameGeometry", err)
	}
}

func TestFrameClearColorsAndPadding(t *testing.T) {
	pixels := []byte{0xff, 0xff}
	frame, err := NewFrame(Size{Width: 9, Height: 1}, 2, pixels)
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}
	if err := frame.Clear(Black); err != nil {
		t.Fatalf("Clear(Black) error = %v", err)
	}
	if pixels[0] != 0xff || pixels[1] != 0x80 {
		t.Fatalf("Clear(Black) = % x, want ff 80", pixels)
	}
	if err := frame.Clear(White); err != nil {
		t.Fatalf("Clear(White) error = %v", err)
	}
	if pixels[0] != 0 || pixels[1] != 0 {
		t.Fatalf("Clear(White) = % x, want 00 00", pixels)
	}
}

func TestFrameClearRejectsInvalidColorWithoutMutation(t *testing.T) {
	pixels := []byte{0xaa}
	frame, err := NewFrame(Size{Width: 8, Height: 1}, 1, pixels)
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}
	if err := frame.Clear(Color(9)); !errors.Is(err, ErrColor) {
		t.Fatalf("Clear(invalid color) error = %v, want ErrColor", err)
	}
	if pixels[0] != 0xaa {
		t.Fatalf("Clear(invalid color) changed frame to % x", pixels)
	}
}

func TestFramePixelColors(t *testing.T) {
	frame, err := NewFrame(Size{Width: 8, Height: 1}, 1, []byte{0xff})
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}
	if got := frame.Pixel(-1, 0); got != White {
		t.Fatalf("Pixel(-1, 0) = %v, want White", got)
	}
	if err := frame.SetPixel(0, 0, White); err != nil {
		t.Fatalf("SetPixel(White) error = %v", err)
	}
	if got := frame.Pixel(0, 0); got != White {
		t.Fatalf("Pixel(0, 0) = %v, want White", got)
	}
	if err := frame.SetPixel(0, 0, Color(9)); !errors.Is(err, ErrColor) {
		t.Fatalf("SetPixel(invalid color) error = %v, want ErrColor", err)
	}
}

func TestFrameOperationsAllocateNothing(t *testing.T) {
	var pixels [16]byte
	frame, err := NewFrame(Size{Width: 16, Height: 8}, 2, pixels[:])
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}

	allocations := testing.AllocsPerRun(100, func() {
		_ = frame.Clear(White)
		_ = frame.SetPixel(4, 3, Black)
		_ = frame.Pixel(4, 3)
		_ = frame.Bytes()
	})
	if allocations != 0 {
		t.Fatalf("frame operations allocations = %v, want 0", allocations)
	}
}
