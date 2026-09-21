package geometry

import (
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

func (b *Box) resolveEdges(c style.Computed) error {
	var err error
	metrics := b.reference.metrics(c, false) // CSS percentage spacing uses width.
	b.Margin, err = resolveEdges(c, "margin-", metrics)
	if err != nil {
		return err
	}
	b.Padding, err = resolveEdges(c, "padding-", metrics)
	if err != nil {
		return err
	}
	b.Border, err = resolveEdges(c, "border-", metrics)
	return err
}

func resolveEdges(c style.Computed, prefix string, metrics style.Metrics) (Edges, error) {
	var values [4]float64
	for index, side := range [...]string{"top", "right", "bottom", "left"} {
		name := prefix + side
		if prefix == "border-" {
			if c.Keyword(name+"-style") != "solid" {
				continue
			}
			name += "-width"
		}
		value := c.Length(name)
		if value.Unit == style.Auto {
			continue
		}
		resolved, err := value.Resolve(metrics)
		if err != nil {
			return Edges{}, err
		}
		values[index] = resolved
	}
	return Edges{values[0], values[1], values[2], values[3]}, nil
}

func (b *Box) autoMargins(c style.Computed) {
	// CSS2.2 10.3.9: auto margins on inline-blocks have used value zero.
	if c.Keyword("display") == "inline-block" || c.Keyword("display") == "inline" {
		return
	}
	left := c.Length("margin-left").Unit == style.Auto
	right := c.Length("margin-right").Unit == style.Auto
	remaining := math.Max(0, b.reference.Width-b.OuterWidth()-b.Margin.Horizontal())
	switch {
	case left && right:
		b.Margin.Left, b.Margin.Right = remaining/2, remaining/2
	case left:
		b.Margin.Left = remaining
	case right:
		b.Margin.Right = remaining
	}
}

func (e Edges) valid() bool {
	return bounded(e.Top) && bounded(e.Right) && bounded(e.Bottom) && bounded(e.Left)
}
