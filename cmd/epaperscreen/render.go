package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"time"
	_ "time/tzdata"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

type renderConfig struct {
	size       display.Size
	zone       *time.Location
	path, port string
}

type rendered struct {
	frame    display.Frame
	warnings []renderdiag.Warning
}

var wallNow = time.Now

func renderHTML(ctx context.Context, c renderConfig, source []byte) (rendered, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tracker, err := refreshstamp.New(c.zone)
	if err != nil {
		return rendered{}, err
	}
	// Capture host time only after device readiness; this is the start of the
	// full cycle, never evidence that writing/compilation refreshed the panel.
	label, err := tracker.Begin(1, wallNow())
	if err != nil {
		return rendered{}, err
	}
	area, err := refreshstamp.Bounds(c.size)
	if err != nil {
		return rendered{}, err
	}
	r, err := engine.New()
	if err != nil {
		return rendered{}, err
	}
	frame, warnings, err := r.RenderReserved(ctx, c.size, source, area)
	if err != nil {
		return rendered{}, err
	}
	if err := refreshstamp.Paint(frame, label); err != nil {
		return rendered{}, err
	}
	return rendered{frame, warnings}, nil
}

func renderOptions(args []string) (renderConfig, error) {
	c := renderConfig{}
	f := flag.NewFlagSet("epaperscreen", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.IntVar(&c.size.Width, "width", 800, "logical display width")
	f.IntVar(&c.size.Height, "height", 480, "logical display height")
	zone := f.String("timezone", "Europe/Kiev", "full-refresh timestamp timezone")
	if err := f.Parse(args); err != nil {
		return renderConfig{}, err
	}
	size := c.size
	if f.NArg() != 2 || size.Width <= 0 || size.Height <= 0 || size.Width > 2048 || size.Height > 2048 || size.Width*size.Height > 1_048_576 {
		return renderConfig{}, errors.New("usage: epaperscreen [-width N -height N -timezone Europe/Kiev] FILE.html SERIAL_PORT (EPS2 only)")
	}
	if *zone == "" {
		return renderConfig{}, refreshstamp.ErrConfiguration
	}
	var err error
	c.zone, err = time.LoadLocation(*zone)
	if err != nil {
		return renderConfig{}, refreshstamp.ErrConfiguration
	}
	c.path, c.port = f.Arg(0), f.Arg(1)
	return c, nil
}

func readHTML(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, errors.Join(err, f.Close())
	}
	if !stat.Mode().IsRegular() {
		return nil, errors.Join(errors.New("HTML input must be a regular file"), f.Close())
	}
	data, err := io.ReadAll(io.LimitReader(f, 32769))
	err = errors.Join(err, f.Close())
	if err != nil {
		return nil, err
	}
	if len(data) > 32768 {
		return nil, &renderdiag.Error{Code: renderdiag.InputLimit, Source: renderdiag.HTML}
	}
	return data, nil
}
