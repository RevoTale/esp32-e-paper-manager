// Package style parses the engine's bounded inline CSS profile into typed values.
// It is manager-only and does not add parsing dependencies to TinyGo firmware.
package style

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

var ErrValue = errors.New("style: invalid or unsupported value")

type Unit uint8

const (
	Px Unit = iota
	Percent
	Em
	Rem
	VW
	VH
	Auto
	None
)

type Length struct {
	Value float64
	Unit  Unit
}

// Metrics supplies the reference for this property, not a global percentage base.
// For font-size, Font is the inherited size (including root's initial size).
// https://www.w3.org/TR/css-values-3/#font-relative-lengths
type Metrics struct {
	Containing, Font, RootFont, ViewportWidth, ViewportHeight float64
}

func ParseLength(source string) (Length, error) {
	value := strings.ToLower(source)
	if value == "auto" {
		return Length{Unit: Auto}, nil
	}
	if value == "none" {
		return Length{Unit: None}, nil
	}
	lexer := css.NewLexer(parse.NewInputString(value))
	kind, data := lexer.Next()
	if next, _ := lexer.Next(); next != css.ErrorToken || string(data) != value {
		return Length{}, ErrValue
	}
	if kind != css.DimensionToken && kind != css.PercentageToken && kind != css.NumberToken {
		return Length{}, ErrValue
	}
	return numericLength(value)
}

func numericLength(value string) (Length, error) {
	units := []struct {
		name string
		unit Unit
	}{{"rem", Rem}, {"px", Px}, {"%", Percent}, {"em", Em}, {"vw", VW}, {"vh", VH}}
	for _, candidate := range units {
		if strings.HasSuffix(value, candidate.name) {
			number, err := boundedNumber(strings.TrimSuffix(value, candidate.name), 8192)
			return Length{Value: number, Unit: candidate.unit}, err
		}
	}
	number, err := boundedNumber(value, 8192)
	if err != nil || number != 0 {
		return Length{}, ErrValue
	}
	return Length{}, nil
}

func boundedNumber(value string, maximum float64) (float64, error) {
	if len(value) == 0 || parse.Number([]byte(value)) != len(value) {
		return 0, ErrValue
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || math.Abs(number) > maximum {
		return 0, ErrValue
	}
	return number, nil
}

func (l Length) Resolve(metrics Metrics) (float64, error) {
	factors := [...]float64{1, metrics.Containing / 100, metrics.Font, metrics.RootFont, metrics.ViewportWidth / 100, metrics.ViewportHeight / 100}
	if int(l.Unit) >= len(factors) {
		return 0, ErrValue
	}
	result := l.Value * factors[l.Unit]
	if math.IsNaN(result) || math.IsInf(result, 0) || math.Abs(result) > 32768 {
		return 0, ErrValue
	}
	return result, nil
}
