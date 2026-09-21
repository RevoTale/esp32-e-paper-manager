package panel

import (
	"errors"
	"testing"
)

func executeStream(s *Stream) error {
	if err := s.Begin(); err != nil {
		return err
	}
	var row [rowBytes]byte
	for pass := uint8(0); pass < 2; pass++ {
		for offset := uint32(0); offset < FrameBytes; offset += rowBytes {
			if err := s.Write(pass, offset, row[:]); err != nil {
				return err
			}
		}
	}
	return s.Commit()
}

func TestEverySPIWriteFailureReleasesStreamingPower(t *testing.T) {
	broken := errors.New("SPI failure")
	// Include every initialization, plane data, refresh and sleep write.
	for failAt := 1; ; failAt++ {
		r := new(recordingIO)
		io := r.io()
		writes := 0
		io.Write = func([]byte) error {
			writes++
			if writes == failAt {
				return broken
			}
			return nil
		}
		d, s := streamFixture(t, io)
		err := executeStream(s)
		if abortErr := s.Abort(); abortErr != nil {
			t.Fatal(abortErr)
		}
		last := r.events[len(r.events)-1]
		if last.kind != "power" || last.high || d.active != 0 {
			t.Fatalf("write %d leaked power", failAt)
		}
		if writes < failAt {
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		if !errors.Is(err, broken) {
			t.Fatalf("write %d: %v", failAt, err)
		}
	}
}

func TestStreamRejectsInvalidChunkWithoutSPI(t *testing.T) {
	if _, err := NewStream(nil); !errors.Is(err, ErrConfig) {
		t.Fatal(err)
	}
	r := new(recordingIO)
	_, s := streamFixture(t, r.io())
	streamError(t, s.Write(0, 0, []byte{0}), ErrStreamState)
	streamError(t, s.Begin(), nil)
	streamError(t, s.Begin(), ErrInUse)
	before := len(r.events)
	for _, p := range [][]byte{nil, make([]byte, rowBytes+1)} {
		streamError(t, s.Write(0, 0, p), ErrStreamState)
	}
	streamError(t, s.Write(1, 0, []byte{0}), ErrStreamState)
	streamError(t, s.Write(0, 1, []byte{0}), ErrStreamState)
	if len(r.events) != before {
		t.Fatal("invalid chunk touched SPI")
	}
	streamError(t, s.Abort(), nil)
	if validStreamChunk(2, 0, 1) || validStreamChunk(0, FrameBytes, 1) {
		t.Fatal("plane bounds")
	}
}

func streamFixture(t *testing.T, io IO) (*Driver, *Stream) {
	t.Helper()
	d, err := New(io)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStream(d)
	if err != nil {
		t.Fatal(err)
	}
	return d, s
}

func streamError(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
