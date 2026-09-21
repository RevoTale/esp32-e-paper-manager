package style

import "github.com/RevoTale/esp32-e-paper-manager/renderdiag"

var sides = [...]string{"top", "right", "bottom", "left"}

var borderParsers = map[string]valueParser{
	"width": borderWidth, "style": borderStyle, "color": colorValue,
}

func borderStyle(raw string) (Value, error) { return keywordValue(raw, "solid none") }

func borderWidth(raw string) (Value, error) {
	value, err := positiveLength(raw)
	if value.Length.Unit == Percent {
		return Value{}, ErrValue
	}
	return value, err
}

func borderProperty(name, raw string) (map[string]Value, renderdiag.Code) {
	if name == "border" {
		return borderShorthand(sides[:], raw)
	}
	for _, side := range sides {
		if name == "border-"+side {
			return borderShorthand([]string{side}, raw)
		}
		for kind, parser := range borderParsers {
			if name == "border-"+side+"-"+kind {
				return singleProperty(name, raw, parser)
			}
		}
	}
	for kind, parser := range borderParsers {
		if name == "border-"+kind {
			return borderFour(kind, raw, parser)
		}
	}
	return nil, renderdiag.UnknownProperty
}

func borderFour(kind, raw string, parser valueParser) (map[string]Value, renderdiag.Code) {
	parts := components(raw)
	if len(parts) == 0 || len(parts) > 4 {
		return nil, renderdiag.InvalidValue
	}
	indices := fourIndices(len(parts))
	values := make(map[string]Value)
	for index, side := range sides {
		value, err := parser(parts[indices[index]])
		if err != nil {
			return nil, renderdiag.InvalidValue
		}
		values["border-"+side+"-"+kind] = value
	}
	return values, 0
}

func borderShorthand(selected []string, raw string) (map[string]Value, renderdiag.Code) {
	parts := components(raw)
	if len(parts) == 0 || len(parts) > 3 {
		return nil, renderdiag.InvalidValue
	}
	values := map[string]Value{
		"width": {Kind: LengthValue, Length: Length{Value: 3}},
		"style": {Kind: KeywordValue, Keyword: "none"},
		"color": {Kind: ColorValue, Keyword: "currentcolor"},
	}
	seen := make(map[string]bool)
	for _, part := range parts {
		kind, value := borderComponent(part)
		if kind == "" || seen[kind] {
			return nil, renderdiag.InvalidValue
		}
		seen[kind], values[kind] = true, value
	}
	result := make(map[string]Value)
	for _, side := range selected {
		for kind, value := range values {
			result["border-"+side+"-"+kind] = value
		}
	}
	return result, 0
}

func borderComponent(raw string) (string, Value) {
	for _, kind := range [...]string{"width", "style", "color"} {
		if value, err := borderParsers[kind](raw); err == nil {
			return kind, value
		}
	}
	return "", Value{}
}

func fourIndices(count int) [4]int {
	indices := [4]int{}
	if count >= 2 {
		indices[1], indices[3] = 1, 1
	}
	if count >= 3 {
		indices[2] = 2
	}
	if count == 4 {
		indices[3] = 3
	}
	return indices
}
