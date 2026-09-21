package panel

import (
	"testing"
	"time"
)

func TestWaitIdleImmediateIdle(t *testing.T) {
	reads := 0
	var events []Event
	driver := Driver{io: IO{
		ReadBusy: func() bool { reads++; return true },
		Delay:    func(time.Duration) { t.Fatal("unexpected delay") },
		Observe:  func(event Event) { events = append(events, event) },
	}}
	if err := driver.waitIdle(PhasePowerOn, StepControllerPowerOn, 0x04, 25*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if reads != 1 {
		t.Fatalf("BUSY reads = %d", reads)
	}
	if len(events) != 2 || events[0].Kind != EventBusyWait || events[1].Kind != EventBusyDone || events[1].Samples != 1 || events[1].LowSamples != 0 {
		t.Fatalf("events = %#v", events)
	}
}
