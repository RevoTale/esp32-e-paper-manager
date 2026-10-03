package main

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/engine"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
)

// Synthetic preview never calls Complete: writing a file confirms no device.
// Use the manager's same reservation, quantization and shared stamp painter.
func runStamped(ctx context.Context, c config, renderer *engine.Renderer, markup []byte, diagnostics io.Writer) error {
	if c.format == "rgba" {
		return errors.New("engine-preview: cycle-start requires mono or frame output")
	}
	label, err := syntheticLabel(c.cycleStart, c.timezone)
	if err != nil {
		return err
	}
	area, err := refreshstamp.Bounds(c.size)
	if err != nil {
		return err
	}
	frame, warnings, err := renderer.RenderReserved(ctx, c.size, markup, area)
	if err != nil {
		return err
	}
	if err := refreshstamp.Paint(frame, label); err != nil {
		return err
	}
	if err := writeWarnings(diagnostics, warnings); err != nil {
		return err
	}
	return writeFrame(c, frame)
}

func syntheticLabel(instant, zoneName string) (refreshstamp.Label, error) {
	if zoneName == "" {
		return refreshstamp.Label{}, refreshstamp.ErrConfiguration
	}
	started, err := time.Parse(time.RFC3339, instant)
	if err != nil {
		return refreshstamp.Label{}, refreshstamp.ErrTime
	}
	zone, err := time.LoadLocation(zoneName)
	if err != nil {
		return refreshstamp.Label{}, refreshstamp.ErrConfiguration
	}
	tracker, err := refreshstamp.New(zone)
	if err != nil {
		return refreshstamp.Label{}, err
	}
	return tracker.Begin(1, started)
}
