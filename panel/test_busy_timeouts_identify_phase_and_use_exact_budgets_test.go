package panel

import (
	"testing"
)

func TestBusyTimeoutsIdentifyPhaseAndUseExactBudgets(t *testing.T) {
	tests := []struct {
		name          string
		targetPhase   Phase
		step          Step
		command       byte
		expectedReads int
	}{
		{name: "power on", targetPhase: PhasePowerOn, step: StepControllerPowerOn, command: 0x04, expectedReads: 2001},
		{name: "refresh", targetPhase: PhaseRefresh, step: StepDisplayRefresh, command: 0x12, expectedReads: 6002},
		{name: "power off", targetPhase: PhasePowerOff, step: StepControllerPowerOff, command: 0x02, expectedReads: 2003},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &recordingIO{}
			io := recorder.io()
			reads := 0
			currentPhase := Phase(0)
			io.Observe = func(event Event) {
				if event.Kind == EventBusyWait {
					currentPhase = event.Phase
				}
			}
			io.ReadBusy = func() bool {
				reads++
				return currentPhase != tc.targetPhase
			}
			driver, err := New(io)
			if err != nil {
				t.Fatal(err)
			}
			err = driver.Refresh(make([]byte, FrameBytes))
			assertBusyOperation(t, err, tc.targetPhase, tc.step, tc.command)
			if code := ErrorCode(err); code != "E_BUSY_TIMEOUT" {
				t.Fatalf("error code = %q", code)
			}
			if reads != tc.expectedReads {
				t.Fatalf("BUSY reads = %d, want %d", reads, tc.expectedReads)
			}
			last := recorder.events[len(recorder.events)-1]
			if last.kind != "power" || last.high {
				t.Fatal("timeout did not end with PWR LOW")
			}
		})
	}
}
