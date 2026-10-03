package panel

import (
	"errors"
	"testing"
)

func TestNewAndFrameValidationDoNotTouchHardware(t *testing.T) {
	base := noOpIO()
	nilCases := []func(*IO){
		func(io *IO) { io.Write = nil },
		func(io *IO) { io.SetCS = nil },
		func(io *IO) { io.SetDC = nil },
		func(io *IO) { io.SetReset = nil },
		func(io *IO) { io.SetPower = nil },
		func(io *IO) { io.ReadBusy = nil },
		func(io *IO) { io.Delay = nil },
	}
	for i, clear := range nilCases {
		io := base
		clear(&io)
		if _, err := New(io); !errors.Is(err, ErrConfig) {
			t.Fatalf("nil callback %d error = %v", i, err)
		}
	}

	touches := 0
	io := base
	io.SetPower = func(bool) { touches++ }
	driver, err := New(io)
	if err != nil {
		t.Fatal(err)
	}
	if err := driver.Refresh(make([]byte, FrameBytes-1)); !errors.Is(err, ErrFrameSize) {
		t.Fatalf("short frame error = %v", err)
	}
	if touches != 0 {
		t.Fatalf("invalid frame touched hardware %d times", touches)
	}
}
