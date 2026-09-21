package panel

import (
	"errors"
	"testing"
)

func TestSPIErrorsIdentifyPhaseAndPowerDown(t *testing.T) {
	injected := errors.New("injected SPI error")
	tests := []struct {
		name    string
		command byte
		row     int
		phase   Phase
		step    Step
		offset  int
	}{
		{name: "initialize command", command: 0x01, row: -1, phase: PhaseInitialize, step: StepPowerSettings, offset: -1},
		{name: "old plane third row", command: 0x10, row: 2, phase: PhaseOldPlane, step: StepOldPlane, offset: 200},
		{name: "new plane first row", command: 0x13, row: 0, phase: PhaseNewPlane, step: StepNewPlane, offset: 0},
		{name: "refresh command", command: 0x12, row: -1, phase: PhaseRefresh, step: StepDisplayRefresh, offset: -1},
		{name: "deep sleep data", command: 0x07, row: 0, phase: PhasePowerOff, step: StepDeepSleep, offset: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &recordingIO{}
			io := recorder.io()
			io = failingSPI(io, tc.command, tc.row, injected)
			driver, err := New(io)
			if err != nil {
				t.Fatal(err)
			}
			err = driver.Refresh(make([]byte, FrameBytes))
			assertOperation(t, err, injected, tc.phase, tc.step, tc.command, tc.offset)
			if code := ErrorCode(err); code != "E_SPI_WRITE" {
				t.Fatalf("error code = %q", code)
			}
			last := recorder.events[len(recorder.events)-1]
			if last.kind != "power" || last.high {
				t.Fatal("failure did not end with PWR LOW")
			}
			if !lastPinLevel(recorder.events, "cs") {
				t.Fatal("failure did not leave CS HIGH")
			}
		})
	}
}
