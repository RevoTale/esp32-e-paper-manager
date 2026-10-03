package geometry

import (
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

// IntrinsicContribution includes fixed dimensions and box edges before an
// ancestor's shrink-to-fit decision. Cyclic percentage sizes behave as auto;
// cyclic percentage margins/padding contribute zero until final layout.
// https://www.w3.org/TR/css-sizing-3/#cyclic-percentage-contribution
func IntrinsicContribution(c style.Computed, ref Reference, content Intrinsic) (Intrinsic, error) {
	ref.Width = 0
	b := Box{reference: ref}
	if err := b.resolveEdges(c); err != nil {
		return Intrinsic{}, err
	}
	extra := b.Padding.Horizontal() + b.Border.Horizontal()
	minWidth, maxWidth, err := intrinsicConstraints(c, ref, extra)
	if err != nil {
		return Intrinsic{}, err
	}
	if value := c.Length("width"); value.Unit < style.Auto && value.Unit != style.Percent && c.Keyword("display") != "inline" {
		width, err := value.Resolve(ref.metrics(c, false))
		if err != nil {
			return Intrinsic{}, err
		}
		if c.Keyword("box-sizing") == "border-box" {
			width = math.Max(0, width-extra)
		}
		content = Intrinsic{width, width}
	}
	extra += b.Margin.Horizontal()
	minimum := math.Max(0, math.Min(maxWidth, math.Max(minWidth, content.Minimum))+extra)
	preferred := math.Max(0, math.Min(maxWidth, math.Max(minWidth, content.Preferred))+extra)
	result := Intrinsic{minimum, math.Max(minimum, preferred)}
	if !validIntrinsic(result) {
		return Intrinsic{}, style.ErrValue
	}
	return result, nil
}

func intrinsicConstraints(c style.Computed, ref Reference, extra float64) (float64, float64, error) {
	minimum, maximum := 0.0, math.Inf(1)
	for _, item := range []struct {
		name string
		out  *float64
	}{{"min-width", &minimum}, {"max-width", &maximum}} {
		value := c.Length(item.name)
		if value.Unit == style.Percent || value.Unit >= style.Auto || c.Keyword("display") == "inline" {
			continue
		}
		resolved, err := value.Resolve(ref.metrics(c, false))
		if err != nil {
			return 0, 0, err
		}
		*item.out = resolved
	}
	if minimum > maximum {
		return 0, 0, style.ErrValue
	}
	if c.Keyword("box-sizing") == "border-box" {
		minimum, maximum = math.Max(0, minimum-extra), math.Max(0, maximum-extra)
	}
	return minimum, maximum, nil
}
