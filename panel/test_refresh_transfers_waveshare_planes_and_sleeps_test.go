package panel

import (
	"bytes"
	"testing"
)

func TestRefreshTransfersWavesharePlanesAndSleeps(t *testing.T) {
	recorder := &recordingIO{}
	driver, err := New(recorder.io())
	if err != nil {
		t.Fatal(err)
	}
	frame := make([]byte, FrameBytes)
	for i := range frame {
		frame[i] = byte(i*29 + 11)
	}
	if err := driver.Refresh(frame); err != nil {
		t.Fatal(err)
	}

	commands, data := commandAndDataWrites(t, recorder.events)
	wantCommands := []byte{0x01, 0x06, 0x04, 0x00, 0x61, 0x15, 0x50, 0x60, 0x10, 0x13, 0x12, 0x50, 0x02, 0x07}
	if !bytes.Equal(commands, wantCommands) {
		t.Fatalf("commands = %x, want %x", commands, wantCommands)
	}
	inverted := make([]byte, len(frame))
	for i, value := range frame {
		inverted[i] = ^value
	}
	if got := data[0x10]; !bytes.Equal(got, inverted) {
		t.Fatal("old plane is not the inverse of the project frame")
	}
	if got := data[0x13]; !bytes.Equal(got, frame) {
		t.Fatal("new plane differs from the project frame")
	}
	last := recorder.events[len(recorder.events)-1]
	if last.kind != "power" || last.high {
		t.Fatal("final event did not disable HAT power")
	}
}
