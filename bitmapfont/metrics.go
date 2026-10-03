// Package bitmapfont defines deterministic metrics shared by layout and raster.
package bitmapfont

import "unicode/utf8"

const (
	BaseWidth       = 6
	BaseHeight      = 13
	BaseAdvance     = 7
	BaseLineHeight  = 15
	TimestampSize   = 13
	TimestampLength = 16
	TimestampPad    = 4
)

type Metrics struct {
	Scale      int
	Width      int
	Height     int
	Advance    int
	LineHeight int
}

func ForSize(size uint8) Metrics {
	height := int(size)
	if height < 1 {
		height = 1
	}
	return Metrics{
		Scale: 1, Width: scaled(BaseWidth, height), Height: height,
		Advance: scaled(BaseAdvance, height), LineHeight: scaled(BaseLineHeight, height),
	}
}

func scaled(value, height int) int { return (value*height + BaseHeight - 1) / BaseHeight }

func GlyphCount(text []byte) int {
	count := 0
	for offset := 0; offset < len(text); count++ {
		_, offset = NextGlyph(text, offset)
	}
	return count
}

func NextGlyph(text []byte, offset int) (rune, int) {
	if offset < 0 || offset >= len(text) {
		return utf8.RuneError, len(text)
	}
	if text[offset] == '&' {
		return entityGlyph(text, offset)
	}
	value, size := utf8.DecodeRune(text[offset:])
	offset += size
	if whitespaceRune(value) {
		for offset < len(text) {
			next, nextSize := utf8.DecodeRune(text[offset:])
			if !whitespaceRune(next) {
				break
			}
			offset += nextSize
		}
		value = ' '
	}
	return value, offset
}

func entityGlyph(text []byte, offset int) (rune, int) {
	start := offset + 1
	end := start
	for end < len(text) && text[end] != ';' {
		end++
	}
	if end == len(text) {
		return utf8.RuneError, len(text)
	}
	name := text[start:end]
	values := [...]struct {
		name string
		char rune
	}{{"amp", '&'}, {"lt", '<'}, {"gt", '>'}, {"quot", '"'}, {"apos", '\''}, {"nbsp", ' '}}
	for _, value := range values {
		if equalBytes(name, value.name) {
			return value.char, end + 1
		}
	}
	return '\ufffd', end + 1
}

func equalBytes(left []byte, right string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func whitespaceRune(value rune) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\r'
}

func RunePixel(value rune, x, y int, metrics Metrics) bool {
	if x < 0 || y < 0 || x >= metrics.Width || y >= metrics.Height {
		return false
	}
	index := 95
	if value >= 0x20 && value < 0x7f {
		index = int(value - 0x20)
	}
	sourceX := x * BaseWidth / metrics.Width
	sourceY := y * BaseHeight / metrics.Height
	return fontMaskPixel(index, sourceX, sourceY)
}

func fontMaskPixel(index, x, y int) bool {
	_, _, _, alpha := fontMaskAt(x, index*BaseHeight+y)
	return alpha != 0
}

func WrappedLines(text []byte, width int, size uint8) int {
	metrics := ForSize(size)
	perLine := width / metrics.Advance
	if perLine < 1 {
		perLine = 1
	}
	glyphs := GlyphCount(text)
	if glyphs == 0 {
		return 0
	}
	return (glyphs + perLine - 1) / perLine
}

func TimestampDimensions() (int, int) {
	metrics := ForSize(TimestampSize)
	return TimestampLength*metrics.Advance + 2*TimestampPad,
		metrics.Height + 2*TimestampPad
}
