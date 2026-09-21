package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/frameio"
)

func previewPaths(t *testing.T, html string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	input := filepath.Join(dir, "input.html")
	if err := os.WriteFile(input, []byte(html), 0600); err != nil {
		t.Fatal(err)
	}
	return input, filepath.Join(dir, "output")
}

func TestNativePreviewFormats(t *testing.T) {
	for _, format := range []string{"rgba", "mono", "frame"} {
		input, output := previewPaths(t, `<div style="width:9px;height:9px;background:black"></div>`)
		var diagnostics bytes.Buffer
		if err := run([]string{"-width", "17", "-height", "9", "-format", format, input, output}, &diagnostics); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		if format == "frame" {
			frame, err := frameio.Decode(data)
			if err != nil || frame.Pixel(0, 0) != display.Black || frame.Pixel(16, 8) != display.White {
				t.Fatalf("frame %v", err)
			}
		} else {
			checkPNG(t, data)
		}
	}
}

func checkPNG(t *testing.T, data []byte) {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil || img.Bounds().Dx() != 17 || img.Bounds().Dy() != 9 {
		t.Fatalf("png %v", err)
	}
}

func TestPreviewRejectsInputsAndDoesNotOverwrite(t *testing.T) {
	input, output := previewPaths(t, "<p>ok</p>")
	for _, args := range [][]string{nil, {"-bad"}, {"-format", "bad", input, output}, {"-width", "0", input, output}, {"/missing-input", output}, {input, "/missing-parent/output"}} {
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
	if err := os.WriteFile(output, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{input, output}, &bytes.Buffer{}); err == nil {
		t.Fatal("overwrote existing file")
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "existing" {
		t.Fatal("changed existing output")
	}
}
