package strictjson

// JSON syntax was validated first, so every escape has its complete payload.
func validEscapes(source []byte) bool {
	for i := 0; i < len(source); i++ {
		if source[i] != '\\' {
			continue
		}
		i++
		if source[i] != 'u' {
			continue
		}
		code := hexUnit(source[i+1 : i+5])
		i += 4
		if code >= 0xdc00 && code <= 0xdfff {
			return false
		}
		if code < 0xd800 || code > 0xdbff {
			continue
		}
		if !lowSurrogate(source[i+1:]) {
			return false
		}
		i += 6
	}
	return true
}

func lowSurrogate(source []byte) bool {
	if len(source) < 6 || source[0] != '\\' || source[1] != 'u' {
		return false
	}
	low := hexUnit(source[2:6])
	return low >= 0xdc00 && low <= 0xdfff
}

func hexUnit(digits []byte) uint16 {
	var code uint16
	for _, digit := range digits {
		var value byte
		switch {
		case digit >= 'a':
			value = digit - 'a' + 10
		case digit >= 'A':
			value = digit - 'A' + 10
		default:
			value = digit - '0'
		}
		code = code*16 + uint16(value)
	}
	return code
}
