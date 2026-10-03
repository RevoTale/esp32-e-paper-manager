package style

import (
	"encoding/hex"
	"image/color"
	"math"
	"strings"

	"golang.org/x/image/colornames"
)

// colorValue keeps straight alpha. Premultiplication occurs at the painter
// boundary, after CSS values have been resolved against inherited currentColor.
// https://www.w3.org/TR/css-color-3/#rgba-color
func colorValue(raw string) (Value, error) {
	raw = strings.ToLower(raw)
	if raw == "currentcolor" {
		return Value{Kind: ColorValue, Keyword: raw}, nil
	}
	if raw == "transparent" {
		return Value{Kind: ColorValue}, nil
	}
	if raw == "rebeccapurple" {
		return Value{Kind: ColorValue, Color: color.NRGBA{102, 51, 153, 255}}, nil
	}
	if named, ok := colornames.Map[raw]; ok {
		return Value{Kind: ColorValue, Color: color.NRGBA(named)}, nil
	}
	var result color.NRGBA
	var err error
	if strings.HasPrefix(raw, "#") {
		result, err = hexColor(raw[1:])
	} else {
		result, err = functionalColor(raw)
	}
	return Value{Kind: ColorValue, Color: result}, err
}

func hexColor(raw string) (color.NRGBA, error) {
	if len(raw) == 3 || len(raw) == 4 {
		var expanded strings.Builder
		for _, digit := range raw {
			expanded.WriteRune(digit)
			expanded.WriteRune(digit)
		}
		raw = expanded.String()
	}
	if len(raw) != 6 && len(raw) != 8 {
		return color.NRGBA{}, ErrValue
	}
	data, err := hex.DecodeString(raw)
	if err != nil {
		return color.NRGBA{}, ErrValue
	}
	result := color.NRGBA{data[0], data[1], data[2], 255}
	if len(data) == 4 {
		result.A = data[3]
	}
	return result, nil
}

func functionalColor(raw string) (color.NRGBA, error) {
	count := 3
	prefix := "rgb("
	if strings.HasPrefix(raw, "rgba(") {
		count, prefix = 4, "rgba("
	}
	if !strings.HasPrefix(raw, prefix) || !strings.HasSuffix(raw, ")") {
		return color.NRGBA{}, ErrValue
	}
	parts := strings.Split(raw[len(prefix):len(raw)-1], ",")
	if len(parts) != count {
		return color.NRGBA{}, ErrValue
	}
	channels := [4]uint8{0, 0, 0, 255}
	percentage := strings.HasSuffix(strings.TrimSpace(parts[0]), "%")
	for index, part := range parts {
		value, err := colorChannel(strings.TrimSpace(part), index == 3, percentage)
		if err != nil {
			return color.NRGBA{}, err
		}
		channels[index] = value
	}
	return color.NRGBA{channels[0], channels[1], channels[2], channels[3]}, nil
}

func colorChannel(raw string, alpha, percentage bool) (uint8, error) {
	maximum := 255.0
	if alpha {
		maximum = 1
	} else if percentage {
		if !strings.HasSuffix(raw, "%") {
			return 0, ErrValue
		}
		raw, maximum = strings.TrimSuffix(raw, "%"), 100
	}
	value, err := boundedNumber(raw, maximum)
	if err != nil || value < 0 {
		return 0, ErrValue
	}
	return uint8(math.Round(value / maximum * 255)), nil
}
