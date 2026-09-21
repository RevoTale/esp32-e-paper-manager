package network

import (
	"bytes"
	"errors"
	"testing"
)

func TestCounterReaderFillsPartialWordAndVaries(t *testing.T) {
	reader, err := NewCounterReader(func() (uint32, error) { return 0x01020304, nil }, 0)
	if err != nil {
		t.Fatal(err)
	}
	first, second := make([]byte, 6), make([]byte, 6)
	if count, readErr := reader.Read(first); readErr != nil || count != len(first) {
		t.Fatalf("count=%d err=%v", count, readErr)
	}
	_, _ = reader.Read(second)
	if bytes.Equal(first, second) {
		t.Fatal("counter did not vary output")
	}
}

func TestCounterReaderPropagatesSourceFailure(t *testing.T) {
	want := errors.New("rng")
	reader, _ := NewCounterReader(func() (uint32, error) { return 0, want }, 0)
	if count, err := reader.Read(make([]byte, 4)); count != 0 || !errors.Is(err, want) {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if _, err := NewCounterReader(nil, 0); !errors.Is(err, ErrEntropy) {
		t.Fatalf("constructor=%v", err)
	}
	var nilReader *CounterReader
	if _, err := nilReader.Read(nil); !errors.Is(err, ErrEntropy) {
		t.Fatalf("nil reader=%v", err)
	}
}
