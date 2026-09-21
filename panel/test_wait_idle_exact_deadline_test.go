package panel

import (
	"testing"
	"time"
)

func TestWaitIdleExactDeadline(t *testing.T) {
	reads := 0
	var delays []time.Duration
	driver := Driver{io: IO{
		ReadBusy: func() bool { reads++; return reads == 6 },
		Delay:    func(d time.Duration) { delays = append(delays, d) },
	}}
	if err := driver.waitIdle(PhaseRefresh, StepDisplayRefresh, 0x12, 25*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if !durationsEqual(delays, []time.Duration{5 * time.Millisecond, 5 * time.Millisecond, 5 * time.Millisecond, 5 * time.Millisecond, 5 * time.Millisecond}) {
		t.Fatalf("delays = %v", delays)
	}
}
