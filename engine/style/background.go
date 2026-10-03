package style

import (
	"strings"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func backgroundDefaults() map[string]Value {
	return map[string]Value{
		"background-color":      {Kind: ColorValue},
		"background-image":      {Kind: ImageValue},
		"background-repeat":     {Kind: KeywordValue, Keyword: "repeat"},
		"background-position-x": {Kind: LengthValue, Length: Length{Unit: Percent}},
		"background-position-y": {Kind: LengthValue, Length: Length{Unit: Percent}},
		"background-size-x":     {Kind: LengthValue, Length: Length{Unit: Auto}},
		"background-size-y":     {Kind: LengthValue, Length: Length{Unit: Auto}},
	}
}

func backgroundProperty(raw string) (map[string]Value, renderdiag.Code) {
	values := backgroundDefaults()
	seen := make(map[string]bool)
	var position, size []string
	afterSlash := false
	parts := components(raw)
	if len(parts) == 0 {
		return nil, renderdiag.InvalidValue
	}
	for _, part := range parts {
		if part == "/" {
			if afterSlash || len(position) == 0 {
				return nil, renderdiag.InvalidValue
			}
			afterSlash = true
			continue
		}
		name, value := backgroundComponent(part)
		if name != "" {
			if seen[name] {
				return nil, renderdiag.InvalidValue
			}
			seen[name], values[name] = true, value
		} else if afterSlash {
			size = append(size, part)
		} else {
			position = append(position, part)
		}
	}
	return completeBackground(values, position, size, afterSlash)
}

func completeBackground(values map[string]Value, position, size []string, slash bool) (map[string]Value, renderdiag.Code) {
	if len(position) > 0 {
		parsed, code := positionProperty("background-position", strings.Join(position, " "))
		if code != 0 {
			return nil, code
		}
		copyValues(values, parsed)
	}
	if slash {
		parsed, code := sizeProperty(strings.Join(size, " "))
		if code != 0 {
			return nil, code
		}
		copyValues(values, parsed)
	}
	return values, 0
}

func backgroundComponent(raw string) (string, Value) {
	if value, err := imageValue(raw); err == nil {
		return "background-image", value
	}
	if value, err := colorValue(raw); err == nil {
		return "background-color", value
	}
	if value, err := keywordValue(raw, keywordProperties["background-repeat"]); err == nil {
		return "background-repeat", value
	}
	return "", Value{}
}

func copyValues(destination, source map[string]Value) {
	for name, value := range source {
		destination[name] = value
	}
}
