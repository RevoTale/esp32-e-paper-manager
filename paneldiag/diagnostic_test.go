package paneldiag

import (
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/panel"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestBusyAndSPIEvidence(t *testing.T) {
	for _, v := range []struct {
		cause error
		code  uint8
	}{{panel.ErrBusyTimeout, BusyTimeout}, {errors.New("private SPI error"), SPIWrite}} {
		err := panel.OpError{Phase: panel.PhaseRefresh, Step: panel.StepDisplayRefresh, Command: 0x12, Offset: -1, BusyKnown: true, Cause: v.cause}
		got := Describe(errors.Join(err, errors.New("private cleanup")))
		want := screenwire.Diagnostic{Domain: screenwire.DomainPanel, Code: v.code, Phase: 5, Step: 14, Command: 0x12, Offset: -1, BusyKnown: true}
		if got != want {
			t.Fatal(got, want)
		}
	}
}

func TestTypedNonOperationErrorsAndUnknownCause(t *testing.T) {
	for _, v := range []struct {
		cause error
		code  uint8
	}{{panel.ErrConfig, Configuration}, {panel.ErrFrameSize, FrameSize}, {panel.ErrInUse, InUse}, {panel.ErrBusyTimeout, BusyTimeout}} {
		got := Describe(v.cause)
		if got.Code != v.code || got.Domain != screenwire.DomainPanel || got.Offset != -1 {
			t.Fatal(got)
		}
	}
	for _, err := range []error{nil, errors.New("not a panel error")} {
		if got := Describe(err); got != (screenwire.Diagnostic{}) {
			t.Fatal(got)
		}
	}
}
