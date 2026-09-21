package panel

import (
	"errors"
	"testing"
	"time"
)

func TestWaitIdleActiveDeadline(t *testing.T) {
	reads := 0
	driver := Driver{io: IO{
		ReadBusy: func() bool { reads++; return false },
		Delay:    func(time.Duration) {},
	}}
	if err := driver.waitIdle(PhaseRefresh, StepDisplayRefresh, 0x12, 25*time.Millisecond); !errors.Is(err, ErrBusyTimeout) || reads != 6 {
		t.Fatalf("error=%v reads=%d", err, reads)
	}
}
