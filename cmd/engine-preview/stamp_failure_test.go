package main

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

func TestPreviewStampValidationPrecedesOutput(t *testing.T) {
	for _, args := range [][]string{
		{"-cycle-start", "2026-09-06T12:34:00"},
		{"-cycle-start", "0001-01-01T00:00:00Z"},
		{"-cycle-start", syntheticStart, "-timezone", ""},
		{"-cycle-start", syntheticStart, "-timezone", "Unknown/Zone"},
		{"-cycle-start", syntheticStart, "-width", "119"},
		{"-cycle-start", syntheticStart, "-height", "20"},
		{"-cycle-start", syntheticStart, "-format", "rgba"},
	} {
		input, output := previewPaths(t, "")
		args = append(args, input, output)
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted invalid stamp arguments: %v", args)
		}
		if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("invalid stamp created output", err)
		}
		if err := os.WriteFile(output, []byte("existing artifact"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Fatal("accepted invalid stamp with existing output")
		}
		data, err := os.ReadFile(output)
		if err != nil || string(data) != "existing artifact" {
			t.Fatal("invalid stamp changed output", err)
		}
	}
}

type failedDiagnostics struct{}

func (failedDiagnostics) Write([]byte) (int, error) { return 0, errors.New("diagnostic write failed") }

func TestPreviewReservedOverlapWarnsBeforeOutput(t *testing.T) {
	input, output := previewPaths(t, `<div style="width:128px;height:32px;background:black"></div>`)
	args := []string{"-width", "128", "-height", "32", "-cycle-start", syntheticStart, input, output}
	if err := run(args, failedDiagnostics{}); err == nil {
		t.Fatal("ignored diagnostic failure")
	}
	if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("warning failure created output", err)
	}
	var diagnostics bytes.Buffer
	if err := run(args, &diagnostics); err != nil {
		t.Fatal(err)
	}
	if diagnostics.Len() == 0 || bytes.Contains(diagnostics.Bytes(), []byte("background")) {
		t.Fatal("missing source-free reservation warning")
	}
}
