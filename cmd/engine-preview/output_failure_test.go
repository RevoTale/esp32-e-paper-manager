package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine"
)

func TestPreviewPreparationFailuresDoNotCreateOutput(t *testing.T) {
	_, output := previewPaths(t, "")
	c := config{format: "frame", output: output}
	if err := writeOutput(context.Background(), c, engine.Result{}); err == nil {
		t.Fatal("accepted absent rendered image")
	}
	if err := writeFrame(c, display.Frame{}); err == nil {
		t.Fatal("accepted invalid packed frame")
	}
	if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("invalid pixels created output", err)
	}
}

func TestPreviewStampedRenderFailureDoesNotCreateOutput(t *testing.T) {
	input, output := previewPaths(t, `<script>private source</script>`)
	if err := run([]string{"-cycle-start", syntheticStart, input, output}, &bytes.Buffer{}); err == nil {
		t.Fatal("accepted invalid stamped document")
	}
	if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("invalid document created output", err)
	}
}

func TestPreviewFrameOutputKeepsExistingFile(t *testing.T) {
	input, output := previewPaths(t, "")
	if err := os.WriteFile(output, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-format", "frame", input, output}, &bytes.Buffer{}); err == nil {
		t.Fatal("overwrote existing frame file")
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "existing" {
		t.Fatal("changed existing frame file", err)
	}
}
