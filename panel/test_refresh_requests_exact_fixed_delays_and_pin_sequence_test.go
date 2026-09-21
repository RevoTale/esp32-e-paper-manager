package panel

import (
	"bytes"
	"testing"
	"time"
)

func TestRefreshRequestsExactFixedDelaysAndPinSequence(t *testing.T) {
	recorder := &recordingIO{busy: []bool{true, true, true}}
	driver, err := New(recorder.io())
	if err != nil {
		t.Fatal(err)
	}
	if err := driver.Refresh(make([]byte, FrameBytes)); err != nil {
		t.Fatal(err)
	}

	delays, power, reset := delayAndPinEvents(recorder.events)
	wantDelays := []time.Duration{
		100 * time.Millisecond, // HAT power-off discharge interval
		100 * time.Millisecond, // HAT power-on settling interval
		20 * time.Millisecond,  // reset HIGH
		2 * time.Millisecond,   // reset LOW
		20 * time.Millisecond,  // reset HIGH
		100 * time.Millisecond, // controller power-on
		100 * time.Millisecond, // display refresh
	}
	if !durationsEqual(delays, wantDelays) {
		t.Fatalf("delays = %v, want %v", delays, wantDelays)
	}
	if !boolsEqual(power, []bool{false, true, false}) {
		t.Fatalf("power = %v", power)
	}
	if !boolsEqual(reset, []bool{true, true, false, true}) {
		t.Fatalf("reset = %v", reset)
	}

	commands, _ := commandAndDataWrites(t, recorder.events)
	if bytes.Contains(commands, []byte{0x71}) {
		t.Fatal("forbidden 0x71 command was sent")
	}
	assertChipSelect(t, recorder.events)
}
