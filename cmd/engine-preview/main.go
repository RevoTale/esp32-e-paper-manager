// engine-preview renders bounded HTML with the native Go engine. It performs
// local file I/O only: no device connection, manager submission or asset fetch.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
	_ "time/tzdata" // Host preview keeps IANA zones available without system files.

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func main() {
	if err := run(os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type config struct {
	size          display.Size
	format        string
	input, output string
	cycleStart    string
	timezone      string
}

func parse(args []string, diagnostics io.Writer) (config, error) {
	c := config{}
	f := flag.NewFlagSet("engine-preview", flag.ContinueOnError)
	f.SetOutput(diagnostics)
	f.IntVar(&c.size.Width, "width", 800, "logical viewport width")
	f.IntVar(&c.size.Height, "height", 480, "logical viewport height")
	f.StringVar(&c.format, "format", "mono", "mono PNG, rgba PNG or frame (BZM1)")
	f.StringVar(&c.cycleStart, "cycle-start", "", "synthetic cycle start in RFC3339; enables stamp for mono/frame")
	f.StringVar(&c.timezone, "timezone", "Europe/Kiev", "synthetic stamp IANA time zone")
	if err := f.Parse(args); err != nil {
		return config{}, err
	}
	if f.NArg() != 2 || c.format != "mono" && c.format != "rgba" && c.format != "frame" {
		return config{}, errors.New("usage: engine-preview [-width N -height N -format mono|rgba|frame -cycle-start RFC3339 -timezone ZONE] FILE.html NEW_OUTPUT")
	}
	c.input, c.output = f.Arg(0), f.Arg(1)
	return c, nil
}

func run(args []string, diagnostics io.Writer) error {
	c, err := parse(args, diagnostics)
	if err != nil {
		return err
	}
	markup, err := readHTML(c.input)
	if err != nil {
		return err
	}
	r, err := engine.New()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if c.cycleStart != "" {
		return runStamped(ctx, c, r, markup, diagnostics)
	}
	result, err := r.RenderRGBA(ctx, c.size, markup)
	if err != nil {
		return err
	}
	if err := writeWarnings(diagnostics, result.Warnings); err != nil {
		return err
	}
	return writeOutput(ctx, c, result)
}

func writeWarnings(output io.Writer, warnings []renderdiag.Warning) error {
	if len(warnings) == 0 {
		return nil
	}
	return json.NewEncoder(output).Encode(warnings)
}

func readHTML(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	markup, err := io.ReadAll(io.LimitReader(f, 32769))
	return markup, errors.Join(err, f.Close())
}
