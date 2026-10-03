package geometry

import (
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

// ObjectRect implements replaced-image fitting inside its already-sized CSS box.
// The painter must clip to that box, even for cover/none or positioned overflow.
// https://www.w3.org/TR/css-images-3/#the-object-fit
func ObjectRect(area Rect, width, height float64, c style.Computed, reference Reference) (Rect, error) {
	w, h, err := fitSize(area, width, height, c.Keyword("object-fit"))
	if err != nil {
		return Rect{}, err
	}
	return positionImage(area, w, h, "object-position", c, reference)
}

func BackgroundRect(area Rect, width, height float64, c style.Computed, reference Reference) (Rect, error) {
	if !area.Valid() || width <= 0 || height <= 0 {
		return Rect{}, style.ErrValue
	}
	x, y := c.Get("background-size-x"), c.Get("background-size-y")
	if x.Kind == style.KeywordValue {
		w, h, err := fitSize(area, width, height, x.Keyword)
		if err != nil {
			return Rect{}, err
		}
		return positionImage(area, w, h, "background-position", c, reference)
	}
	metrics := reference.metrics(c, false)
	w, h, err := backgroundDimensions(area, width, height, x.Length, y.Length, metrics)
	if err != nil {
		return Rect{}, err
	}
	return positionImage(area, w, h, "background-position", c, reference)
}

// BackgroundPosition places an already-sized image in its fragment's padding
// box. Fragmented backgrounds can share an aspect-derived size across boxes.
func BackgroundPosition(area Rect, width, height float64, c style.Computed, reference Reference) (Rect, error) {
	if !area.Valid() || !nonnegative(width) || !nonnegative(height) {
		return Rect{}, style.ErrValue
	}
	return positionImage(area, width, height, "background-position", c, reference)
}

func fitSize(area Rect, width, height float64, mode string) (float64, float64, error) {
	if !validImageBounds(area, width, height) {
		return 0, 0, style.ErrValue
	}
	ratio := math.Min(area.Width/width, area.Height/height)
	switch mode {
	case "fill":
		return area.Width, area.Height, nil
	case "cover":
		ratio = math.Max(area.Width/width, area.Height/height)
	case "contain":
	case "scale-down":
		ratio = math.Min(1, ratio)
	case "none":
		ratio = 1
	default:
		return 0, 0, style.ErrValue
	}
	return width * ratio, height * ratio, nil
}

func validImageBounds(area Rect, width, height float64) bool {
	return area.Valid() && nonnegative(width) && nonnegative(height) && width > 0 && height > 0
}

func backgroundDimensions(area Rect, width, height float64, x, y style.Length, metrics style.Metrics) (float64, float64, error) {
	w, h := width, height
	var err error
	if x.Unit != style.Auto {
		metrics.Containing = area.Width
		w, err = x.Resolve(metrics)
		if err != nil {
			return 0, 0, err
		}
	}
	if y.Unit != style.Auto {
		metrics.Containing = area.Height
		h, err = y.Resolve(metrics)
		if err != nil {
			return 0, 0, err
		}
	}
	if x.Unit == style.Auto && y.Unit != style.Auto {
		w = h * width / height
	} else if y.Unit == style.Auto && x.Unit != style.Auto {
		h = w * height / width
	}
	return w, h, nil
}

func positionImage(area Rect, width, height float64, property string, c style.Computed, reference Reference) (Rect, error) {
	metrics := reference.metrics(c, false)
	metrics.Containing = area.Width - width
	x, err := c.Length(property + "-x").Resolve(metrics)
	if err != nil {
		return Rect{}, err
	}
	metrics.Containing = area.Height - height
	y, err := c.Length(property + "-y").Resolve(metrics)
	result := Rect{area.X + x, area.Y + y, width, height}
	if err != nil || !result.Valid() {
		return Rect{}, style.ErrValue
	}
	return result, nil
}
