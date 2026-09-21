package style

import (
	"encoding/base64"
	"strings"

	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func positionProperty(name, raw string) (map[string]Value, renderdiag.Code) {
	parts := components(strings.ToLower(raw))
	if len(parts) < 1 || len(parts) > 2 {
		return nil, renderdiag.InvalidValue
	}
	if len(parts) == 1 {
		parts = append(parts, "center")
	}
	if reorderPosition(parts[0], parts[1]) {
		parts[0], parts[1] = parts[1], parts[0]
	}
	x, errX := positionAxis(parts[0], true)
	y, errY := positionAxis(parts[1], false)
	if errX != nil || errY != nil {
		return nil, renderdiag.InvalidValue
	}
	return map[string]Value{name + "-x": x, name + "-y": y}, 0
}

func reorderPosition(first, second string) bool {
	isKeyword := func(value string) bool { return vertical(value) || horizontal(value) || value == "center" }
	return isKeyword(first) && isKeyword(second) && (vertical(first) || horizontal(second))
}

func vertical(value string) bool   { return value == "top" || value == "bottom" }
func horizontal(value string) bool { return value == "left" || value == "right" }

func positionAxis(raw string, x bool) (Value, error) {
	if raw == "center" {
		return Value{Kind: LengthValue, Length: Length{Value: 50, Unit: Percent}}, nil
	}
	if horizontal(raw) || vertical(raw) {
		if horizontal(raw) != x {
			return Value{}, ErrValue
		}
		number := 0.0
		if raw == "right" || raw == "bottom" {
			number = 100
		}
		return Value{Kind: LengthValue, Length: Length{Value: number, Unit: Percent}}, nil
	}
	value, err := ParseLength(raw)
	if err != nil || value.Unit >= Auto {
		return Value{}, ErrValue
	}
	return Value{Kind: LengthValue, Length: value}, nil
}

func sizeProperty(raw string) (map[string]Value, renderdiag.Code) {
	parts := components(strings.ToLower(raw))
	if len(parts) < 1 || len(parts) > 2 {
		return nil, renderdiag.InvalidValue
	}
	if len(parts) == 1 && (parts[0] == "contain" || parts[0] == "cover") {
		value := Value{Kind: KeywordValue, Keyword: parts[0]}
		return map[string]Value{"background-size-x": value, "background-size-y": value}, 0
	}
	if len(parts) == 1 {
		parts = append(parts, "auto")
	}
	values := make(map[string]Value)
	for index, axis := range [...]string{"x", "y"} {
		value, err := propertyLength("width", parts[index])
		if err != nil {
			return nil, renderdiag.InvalidValue
		}
		values["background-size-"+axis] = Value{Kind: LengthValue, Length: value}
	}
	return values, 0
}

func imageValue(raw string) (Value, error) {
	if strings.EqualFold(raw, "none") {
		return Value{Kind: ImageValue}, nil
	}
	if len(raw) < 6 || !strings.EqualFold(raw[:4], "url(") || raw[len(raw)-1] != ')' {
		return Value{}, ErrValue
	}
	url := strings.TrimSpace(raw[4 : len(raw)-1])
	url = unquoteURL(url)
	if strings.ContainsAny(url, "\\\r\n\"'() ") || !canonicalImageURL(url) {
		return Value{}, ErrValue
	}
	return Value{Kind: ImageValue, Image: url}, nil
}

func unquoteURL(value string) string {
	if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') && value[len(value)-1] == value[0] {
		return value[1 : len(value)-1]
	}
	return value
}

func canonicalImageURL(url string) bool {
	for _, prefix := range [...]string{"data:image/png;base64,", "data:image/jpeg;base64,"} {
		if strings.HasPrefix(url, prefix) {
			data := url[len(prefix):]
			decoded, err := base64.StdEncoding.Strict().DecodeString(data)
			return err == nil && len(decoded) > 0 && base64.StdEncoding.EncodeToString(decoded) == data
		}
	}
	return false
}
