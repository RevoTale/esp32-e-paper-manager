package panel

import (
	"testing"
)

func assertChipSelect(t *testing.T, events []wireEvent) {
	t.Helper()
	csHigh := true
	for i, event := range events {
		if event.kind == "cs" {
			csHigh = event.high
		}
		if event.kind == "write" && csHigh {
			t.Fatalf("write %d occurred while CS HIGH", i)
		}
	}
}
