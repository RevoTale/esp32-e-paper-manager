package geometry

import (
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

// CSS2.2 sizing references: https://www.w3.org/TR/CSS22/visudet.html
// Indefinite percentage heights are auto; minima/maxima use their auto fallback.
func resolveAxis(c style.Computed, reference Reference, axis string, extra, automatic float64) (float64, bool, error) {
	metrics := reference.metrics(c, axis == "height")
	minimum, maximum, err := axisConstraints(c, reference, axis, extra)
	if err != nil {
		return 0, false, err
	}
	value, definite, err := axisLength(c, reference, axis, automatic, metrics)
	if err != nil {
		return 0, false, err
	}
	if c.Keyword("box-sizing") == "border-box" && definite {
		value = math.Max(0, value-extra)
	}
	value = math.Min(maximum, math.Max(minimum, value))
	if !nonnegative(value) {
		return 0, false, style.ErrValue
	}
	return value, definite, nil
}

func axisConstraints(c style.Computed, reference Reference, axis string, extra float64) (float64, float64, error) {
	metrics := reference.metrics(c, axis == "height")
	minimum, _, err := axisLength(c, reference, "min-"+axis, 0, metrics)
	if err != nil {
		return 0, 0, err
	}
	maximum, _, err := axisLength(c, reference, "max-"+axis, math.Inf(1), metrics)
	if err != nil || minimum > maximum {
		return 0, 0, style.ErrValue
	}
	if c.Keyword("box-sizing") == "border-box" {
		minimum = math.Max(0, minimum-extra)
		maximum = math.Max(0, maximum-extra)
	}
	return minimum, maximum, nil
}

func axisLength(c style.Computed, ref Reference, name string, fallback float64, metrics style.Metrics) (float64, bool, error) {
	value := c.Length(name)
	if value.Unit >= style.Auto || indefiniteHeight(name, value, ref) {
		return fallback, false, nil
	}
	resolved, err := value.Resolve(metrics)
	if err != nil {
		return 0, false, err
	}
	return resolved, true, nil
}

func indefiniteHeight(name string, value style.Length, ref Reference) bool {
	if ref.DefiniteHeight || value.Unit != style.Percent {
		return false
	}
	return name == "height" || name == "min-height" || name == "max-height"
}
