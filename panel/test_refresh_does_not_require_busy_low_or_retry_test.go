package panel

import (
	"testing"
)

func TestRefreshDoesNotRequireBusyLowOrRetry(t *testing.T) {
	io := noOpIO()
	attempts := 0
	io.SetPower = func(high bool) {
		if high {
			attempts++
		}
	}
	io.ReadBusy = func() bool { return true }
	driver, err := New(io)
	if err != nil {
		t.Fatal(err)
	}

	if err := driver.Refresh(make([]byte, FrameBytes)); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("power-on attempts = %d, want 1", attempts)
	}
}
