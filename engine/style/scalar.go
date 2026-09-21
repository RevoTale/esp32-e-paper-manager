package style

import (
	"math"
	"strings"
)

type valueParser func(string) (Value, error)

var keywordProperties = map[string]string{
	"display": "block inline inline-block none", "position": "static relative absolute",
	"box-sizing": "content-box border-box", "overflow": "visible hidden clip",
	"font-style": "normal italic", "text-align": "left right center justify start end",
	"white-space": "normal nowrap pre pre-wrap pre-line", "overflow-wrap": "normal anywhere",
	"text-overflow": "clip ellipsis", "object-fit": "fill contain cover none scale-down",
	"background-repeat": "repeat no-repeat repeat-x repeat-y",
}

var scalarProperties = map[string]valueParser{
	"color": colorValue, "background-color": colorValue, "opacity": opacityValue,
	"z-index": zIndexValue, "font-family": familyValue, "font-size": fontSizeValue,
	"font-weight": weightValue, "line-height": lineHeightValue,
	"border-radius": radiusValue, "aspect-ratio": ratioValue,
}

func keywordValue(raw, options string) (Value, error) {
	value := strings.ToLower(raw)
	for _, option := range strings.Fields(options) {
		if option == value {
			return Value{Kind: KeywordValue, Keyword: value}, nil
		}
	}
	return Value{}, ErrValue
}

func opacityValue(raw string) (Value, error) {
	number, err := boundedNumber(raw, 1)
	if err != nil || number < 0 {
		return Value{}, ErrValue
	}
	return Value{Kind: NumberValue, Number: number}, nil
}

func zIndexValue(raw string) (Value, error) {
	raw = strings.ToLower(raw)
	if raw == "auto" {
		return Value{Kind: KeywordValue, Keyword: raw}, nil
	}
	number, err := boundedNumber(raw, 32768)
	if err != nil || number > 32767 || math.Trunc(number) != number {
		return Value{}, ErrValue
	}
	return Value{Kind: NumberValue, Number: number}, nil
}

func fontSizeValue(raw string) (Value, error) {
	value, err := positiveLength(raw)
	if err != nil || value.Length.Value == 0 {
		return Value{}, ErrValue
	}
	return value, nil
}

func weightValue(raw string) (Value, error) {
	value, err := keywordValue(raw, "normal bold 400 700")
	switch value.Keyword {
	case "400":
		value.Keyword = "normal"
	case "700":
		value.Keyword = "bold"
	}
	return value, err
}

func lineHeightValue(raw string) (Value, error) {
	raw = strings.ToLower(raw)
	if raw == "normal" {
		return Value{Kind: KeywordValue, Keyword: raw}, nil
	}
	if number, err := boundedNumber(raw, 16); err == nil && number > 0 {
		return Value{Kind: NumberValue, Number: number}, nil
	}
	return fontSizeValue(raw)
}

func positiveLength(raw string) (Value, error) {
	length, err := ParseLength(raw)
	if err != nil || length.Value < 0 || length.Unit >= Auto {
		return Value{}, ErrValue
	}
	return Value{Kind: LengthValue, Length: length}, nil
}

func radiusValue(raw string) (Value, error) { return positiveLength(raw) }

func ratioValue(raw string) (Value, error) {
	raw = strings.ToLower(raw)
	if raw == "auto" {
		return Value{Kind: KeywordValue, Keyword: raw}, nil
	}
	parts := strings.Split(raw, "/")
	if len(parts) != 2 {
		return Value{}, ErrValue
	}
	x, errX := boundedNumber(strings.TrimSpace(parts[0]), 8192)
	y, errY := boundedNumber(strings.TrimSpace(parts[1]), 8192)
	if errX != nil || errY != nil || x <= 0 || y <= 0 {
		return Value{}, ErrValue
	}
	ratio := x / y
	if ratio <= 0 || math.IsInf(ratio, 0) || ratio > 8192 {
		return Value{}, ErrValue
	}
	return Value{Kind: NumberValue, Number: ratio}, nil
}
