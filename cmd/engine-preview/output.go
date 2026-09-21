package main

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine"
	"github.com/RevoTale/esp32-e-paper-manager/frameio"
)

func writeOutput(ctx context.Context, c config, result engine.Result) error {
	if c.format == "rgba" {
		return writePNG(c.output, result.Image)
	}
	frame, err := result.Frame(ctx)
	if err != nil {
		return err
	}
	return writeFrame(c, frame)
}

func writeFrame(c config, frame display.Frame) error {
	if c.format == "mono" {
		return writePNG(c.output, frameImage{frame})
	}
	reader, err := frameio.Reader(frame)
	if err != nil {
		return err
	}
	output, err := os.OpenFile(c.output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, reader)
	return errors.Join(err, output.Close())
}

func writePNG(path string, img image.Image) error {
	output, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	err = png.Encode(output, img)
	return errors.Join(err, output.Close())
}

type frameImage struct{ frame display.Frame }

func (f frameImage) ColorModel() color.Model { return color.GrayModel }
func (f frameImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, f.frame.Size().Width, f.frame.Size().Height)
}
func (f frameImage) At(x, y int) color.Color {
	if f.frame.Pixel(x, y) == display.Black {
		return color.Gray{}
	}
	return color.Gray{Y: 255}
}
