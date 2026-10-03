package style

import (
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

// WithDimensionHints applies HTML image width/height below inline declarations.
// Zero means absent. It preserves CSS source ranges and never changes b.
func (b Block) WithDimensionHints(width, height float64) (Block, error) {
	result := Block{values: make(map[string]declaration, len(b.values)+2), count: b.count, element: b.element}
	for name, entry := range b.values {
		result.values[name] = entry
	}
	for name, value := range map[string]float64{"width": width, "height": height} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 8192 {
			return Block{}, b.Error(name, renderdiag.InvalidValue)
		}
		if _, present := result.values[name]; value > 0 && !present {
			result.values[name] = declaration{value: Value{Kind: LengthValue, Length: Length{Value: value}}, location: renderdiag.Error{Source: renderdiag.HTML, Element: b.element}}
		}
	}
	return result, nil
}
