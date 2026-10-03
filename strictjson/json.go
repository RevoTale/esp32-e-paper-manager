// Package strictjson rejects ambiguous bounded JSON before typed decoding.
package strictjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"unicode/utf8"
)

var ErrInvalid = errors.New("json: invalid or ambiguous document")

// Validate requires one valid UTF-8 JSON value, paired Unicode escapes, unique
// decoded object keys and at most 32 container levels in a 32 KiB document.
// encoding/json otherwise replaces invalid Unicode and accepts duplicate keys:
// https://pkg.go.dev/encoding/json#Unmarshal
func Validate(source []byte) error {
	if len(source) > 32768 || !utf8.Valid(source) || !json.Valid(source) || !validEscapes(source) {
		return ErrInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.UseNumber()
	return value(decoder, 0)
}

func value(decoder *json.Decoder, depth int) error {
	token, err := decoder.Token()
	if err != nil {
		return ErrInvalid
	}
	opening, container := token.(json.Delim)
	if !container {
		return nil
	}
	if depth >= 32 {
		return ErrInvalid
	}
	seen := make(map[string]bool)
	for decoder.More() {
		if opening == '{' {
			key, err := decoder.Token()
			if err != nil {
				return ErrInvalid
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return ErrInvalid
			}
			seen[name] = true
		}
		if err := value(decoder, depth+1); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}
