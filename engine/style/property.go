package style

import (
	"strings"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func property(name, raw string) (map[string]Value, renderdiag.Code) {
	if options, ok := keywordProperties[name]; ok {
		return singleProperty(name, raw, func(value string) (Value, error) { return keywordValue(value, options) })
	}
	if parser, ok := scalarProperties[name]; ok {
		return singleProperty(name, raw, parser)
	}
	return compoundProperty(name, raw)
}

func singleProperty(name, raw string, parser valueParser) (map[string]Value, renderdiag.Code) {
	value, err := parser(raw)
	if err != nil {
		return nil, renderdiag.InvalidValue
	}
	return map[string]Value{name: value}, 0
}

func compoundProperty(name, raw string) (map[string]Value, renderdiag.Code) {
	switch name {
	case "background":
		return backgroundProperty(raw)
	case "border":
		return borderProperty(name, raw)
	case "background-position", "object-position":
		return positionProperty(name, raw)
	case "background-size":
		return sizeProperty(raw)
	case "margin", "padding":
		return spacing(name, raw)
	case "background-image":
		return singleProperty(name, raw, imageValue)
	}
	if strings.HasPrefix(name, "border-") {
		return borderProperty(name, raw)
	}
	if isLengthProperty(name) {
		return singleProperty(name, raw, func(raw string) (Value, error) {
			value, err := propertyLength(name, raw)
			return Value{Kind: LengthValue, Length: value}, err
		})
	}
	return nil, renderdiag.UnknownProperty
}

func isLengthProperty(name string) bool {
	switch name {
	case "width", "height", "min-width", "min-height", "max-width", "max-height", "top", "right", "bottom", "left",
		"margin-top", "margin-right", "margin-bottom", "margin-left", "padding-top", "padding-right", "padding-bottom", "padding-left":
		return true
	}
	return false
}

func propertyLength(name, raw string) (Length, error) {
	value, err := ParseLength(raw)
	if err != nil {
		return Length{}, err
	}
	rule := lengthPolicy(name)
	if value.Unit == None && !rule.none {
		return Length{}, ErrValue
	}
	if value.Unit == Auto && !rule.auto {
		return Length{}, ErrValue
	}
	if value.Value < 0 && !rule.negative {
		return Length{}, ErrValue
	}
	return value, nil
}

type lengthRule struct{ auto, none, negative bool }

func lengthPolicy(name string) lengthRule {
	if strings.HasPrefix(name, "margin-") {
		return lengthRule{auto: true, negative: true}
	}
	switch name {
	case "top", "right", "bottom", "left":
		return lengthRule{auto: true, negative: true}
	case "width", "height":
		return lengthRule{auto: true}
	case "max-width", "max-height":
		return lengthRule{none: true}
	}
	return lengthRule{}
}

func spacing(name, raw string) (map[string]Value, renderdiag.Code) {
	parts := strings.Fields(raw)
	if len(parts) == 0 || len(parts) > 4 {
		return nil, renderdiag.InvalidValue
	}
	indices := fourIndices(len(parts))
	values := make(map[string]Value, 4)
	for index, suffix := range [...]string{"top", "right", "bottom", "left"} {
		key := name + "-" + suffix
		value, err := propertyLength(key, parts[indices[index]])
		if err != nil {
			return nil, renderdiag.InvalidValue
		}
		values[key] = Value{Kind: LengthValue, Length: value}
	}
	return values, 0
}
