package panel

import (
	"bytes"
	"reflect"
	"testing"
)

func TestStreamMatchesBufferedWire(t *testing.T) {
	frame := make([]byte, FrameBytes)
	for i := range frame {
		frame[i] = byte(i*29 + 11)
	}
	buffered, streamed := new(recordingIO), new(recordingIO)
	d, err := New(buffered.io())
	if err != nil {
		t.Fatal(err)
	}
	if err = d.Refresh(frame); err != nil {
		t.Fatal(err)
	}
	d, err = New(streamed.io())
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStream(d)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Begin(); err != nil {
		t.Fatal(err)
	}
	writeStreamFrame(t, s, frame)
	assertNoRefresh(t, streamed)
	if err = s.Commit(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(buffered.events, streamed.events) {
		t.Fatal("stream changed verified wire sequence")
	}
}

func writeStreamFrame(t *testing.T, s *Stream, frame []byte) {
	t.Helper()
	for pass := uint8(0); pass < 2; pass++ {
		for offset := 0; offset < FrameBytes; offset += rowBytes {
			if err := s.Write(pass, uint32(offset), frame[offset:offset+rowBytes]); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func assertNoRefresh(t *testing.T, r *recordingIO) {
	t.Helper()
	commands, _ := commandAndDataWrites(t, r.events)
	if bytes.Contains(commands, []byte{0x12}) {
		t.Fatal("premature refresh")
	}
}

func TestStreamAbortReleasesDriver(t *testing.T) {
	r := new(recordingIO)
	d, err := New(r.io())
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStream(d)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Begin(); err != nil {
		t.Fatal(err)
	}
	if err = d.Refresh(make([]byte, FrameBytes)); err != ErrInUse {
		t.Fatal(err)
	}
	if err = s.Commit(); err == nil {
		t.Fatal("incomplete commit")
	}
	if err = s.Abort(); err != nil {
		t.Fatal(err)
	}
	assertNoRefresh(t, r)
	if err = d.Refresh(make([]byte, FrameBytes)); err != nil {
		t.Fatal(err)
	}
}
