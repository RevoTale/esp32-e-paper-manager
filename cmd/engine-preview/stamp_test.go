package main

import (
	"bytes"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/frameio"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
)

const syntheticStart = "2026-09-06T12:34:00Z"

func expectedStamp(t *testing.T, offset int, text string) display.Frame {
	t.Helper()
	pixels := make([]byte, 16*32)
	for y := range 4 {
		pixels[y*16] = 0xe0
	}
	frame, err := display.NewFrame(display.Size{Width: 128, Height: 32}, 16, pixels)
	if err != nil {
		t.Fatal(err)
	}
	tracker, err := refreshstamp.New(time.FixedZone("expected explicit offset", offset))
	if err != nil {
		t.Fatal(err)
	}
	label, err := tracker.Begin(1, time.Date(2026, 9, 6, 12, 34, 0, 0, time.UTC))
	if err != nil || label.Text() != text {
		t.Fatal("incorrect independent time fixture", err)
	}
	if err := refreshstamp.Paint(frame, label); err != nil {
		t.Fatal(err)
	}
	return frame
}

func TestPreviewSyntheticCycleUsesSharedStamp(t *testing.T) {
	want := expectedStamp(t, 3*3600, "2026-09-06 15:34")
	for _, format := range []string{"mono", "frame"} {
		input, output := previewPaths(t, `<div style="width:3px;height:4px;background:black"></div>`)
		var diagnostics bytes.Buffer
		args := []string{"-width", "128", "-height", "32", "-format", format, "-cycle-start", syntheticStart, input, output}
		if err := run(args, &diagnostics); err != nil {
			t.Fatal(err)
		}
		if diagnostics.Len() != 0 {
			t.Fatal("unexpected warning", diagnostics.String())
		}
		assertStampedOutput(t, output, format, want)
	}
}

func TestPreviewSyntheticUTCOverridesDefaultZone(t *testing.T) {
	input, output := previewPaths(t, `<div style="width:3px;height:4px;background:black"></div>`)
	args := []string{"-width", "128", "-height", "32", "-format", "frame", "-cycle-start", syntheticStart, "-timezone", "UTC", input, output}
	if err := run(args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	assertStampedOutput(t, output, "frame", expectedStamp(t, 0, "2026-09-06 12:34"))
}

func assertStampedOutput(t *testing.T, output, format string, want display.Frame) {
	t.Helper()
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if format == "frame" {
		got, err := frameio.Decode(data)
		if err != nil || !bytes.Equal(got.Bytes(), want.Bytes()) {
			t.Fatal("stamp frame differs", err)
		}
		return
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for y := range want.Size().Height {
		for x := range want.Size().Width {
			r, _, _, _ := img.At(x, y).RGBA()
			if (r == 0) != (want.Pixel(x, y) == display.Black) {
				t.Fatalf("stamp pixel differs at %d,%d", x, y)
			}
		}
	}
}

func TestPreviewSyntheticZoneAndEquivalentInstant(t *testing.T) {
	var baseline []byte
	for _, instant := range []string{syntheticStart, "2026-09-06T15:34:00+03:00"} {
		input, output := previewPaths(t, "")
		args := []string{"-cycle-start", instant, "-timezone", "Europe/Kiev", "-format", "frame", input, output}
		if err := run(args, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		if baseline != nil && !bytes.Equal(baseline, data) {
			t.Fatal("same instant changed output")
		}
		baseline = data
	}
}
