// Package paneldiag maps the tested Waveshare driver to source-free EPS2
// diagnostics. Keep physical phase/step values stable within profile version 1.
package paneldiag

import (
	"errors"

	"github.com/RevoTale/esp32-e-paper-manager/panel"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

const (
	Unknown uint8 = iota + 1
	BusyTimeout
	SPIWrite
	Configuration
	FrameSize
	InUse
)

// Describe preserves errors.As/Is evidence through cleanup errors.Join without
// formatting the underlying error (which may contain device-specific text).
func Describe(err error) screenwire.Diagnostic {
	code := classify(err)
	if code == 0 {
		return screenwire.Diagnostic{}
	}
	d := screenwire.Diagnostic{Domain: screenwire.DomainPanel, Code: code, Offset: -1}
	var op panel.OpError
	if errors.As(err, &op) {
		d.Phase, d.Step, d.Command = uint8(op.Phase), uint8(op.Step), op.Command
		d.BusyKnown, d.Busy = op.BusyKnown, op.Busy && op.BusyKnown
		if op.Offset >= -1 && int64(op.Offset) <= 1<<31-1 {
			d.Offset = int32(op.Offset)
		}
	}
	return d
}

func classify(err error) uint8 {
	switch panel.ErrorCode(err) {
	case "E_BUSY_TIMEOUT":
		return BusyTimeout
	case "E_SPI_WRITE":
		return SPIWrite
	case "E_PANEL_CONFIG":
		return Configuration
	case "E_FRAME_SIZE":
		return FrameSize
	case "E_PANEL_IN_USE":
		return InUse
	default:
		return 0
	}
}
