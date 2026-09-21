package panel

import (
	"errors"
	"testing"
)

func assertOperation(t *testing.T, err, cause error, phase Phase, step Step, command byte, offset int) {
	t.Helper()
	var op OpError
	if !errors.Is(err, cause) || !errors.As(err, &op) ||
		op.Phase != phase || op.Step != step || op.Command != command || op.Offset != offset {
		t.Fatalf("error = %#v, want phase=%s step=%s command=%02x offset=%d", err, phase, step, command, offset)
	}
}
