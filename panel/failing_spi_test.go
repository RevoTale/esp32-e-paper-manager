package panel

func failingSPI(io IO, command byte, row int, injected error) IO {
	dc := false
	currentCommand := byte(0xff)
	dataWrites := 0
	originalSetDC := io.SetDC
	io.SetDC = func(high bool) {
		dc = high
		originalSetDC(high)
	}
	originalWrite := io.Write
	io.Write = func(p []byte) error {
		if !dc {
			currentCommand = p[0]
			dataWrites = 0
			if currentCommand == command && row < 0 {
				return injected
			}
		} else if currentCommand == command {
			if dataWrites == row {
				return injected
			}
			dataWrites++
		}
		return originalWrite(p)
	}

	return io
}
