package panel

import (
	"errors"
	"testing"
)

func assertBusyOperation(t *testing.T, err error, phase Phase, step Step, command byte) {
	t.Helper()
	var op OpError
	if !errors.Is(err, ErrBusyTimeout) || !errors.As(err, &op) ||
		op.Phase != phase || op.Step != step || op.Command != command ||
		!op.BusyKnown || op.Busy {
		t.Fatalf("error = %#v, want timeout phase=%s step=%s command=%02x", err, phase, step, command)
	}
}
