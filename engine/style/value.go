package style

import (
	"image/color"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

type Kind uint8

const (
	LengthValue Kind = iota + 1
	KeywordValue
	NumberValue
	ColorValue
	ImageValue
)

// Value is a closed typed union, validated at Parse. A property's grammar
// determines its Kind; source strings never reach geometry or paint arithmetic.
type Value struct {
	Kind    Kind
	Length  Length
	Number  float64
	Color   color.NRGBA
	Keyword string
	Image   string
}

type declaration struct {
	value     Value
	important bool
	location  renderdiag.Error
}

type Block struct {
	values  map[string]declaration
	count   int
	element uint32
}

func (b Block) Count() int { return b.count }

func (b Block) Get(name string) (Value, bool) {
	entry, ok := b.values[name]
	return entry.value, ok
}

func (b *Block) apply(values map[string]Value, important bool, location renderdiag.Error) {
	for name, value := range values {
		previous, exists := b.values[name]
		if !exists || important || !previous.important {
			b.values[name] = declaration{value: value, important: important, location: location}
		}
	}
}

// Error retains the winning declaration's source range after shorthand/cascade.
// A generated semantic default has HTML identity but no fabricated CSS range.
func (b Block) Error(name string, code renderdiag.Code) error {
	location := renderdiag.Error{Source: renderdiag.HTML, Element: b.element}
	if entry, ok := b.values[name]; ok {
		location = entry.location
	}
	location.Code = code
	return &location
}
