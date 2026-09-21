package streamrx

import (
	"bytes"
	"errors"
	"slices"
	"testing"
)

type chunkSink struct {
	sink
	pass   uint8
	offset uint32
	data   []byte
	sizes  []int
	failAt int
}

func (s *chunkSink) Write(pass uint8, offset uint32, data []byte) error {
	if pass != s.pass || offset != s.offset {
		return ErrState
	}
	s.writes++
	if s.failAt == s.writes {
		return ErrState
	}
	s.data = append(s.data, data...)
	s.sizes = append(s.sizes, len(data))
	s.offset += uint32(len(data))
	return nil
}

func TestChunkedSinkPreservesBytesAndOffsets(t *testing.T) {
	s := &chunkSink{pass: 1, offset: 17}
	c, err := NewChunkedSink(s, 100)
	if err != nil {
		t.Fatal(err)
	}
	pixels := bytes.Repeat([]byte{0x91}, 1024)
	if err = c.Begin(); err != nil {
		t.Fatal(err)
	}
	if err = c.Write(1, 17, pixels); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(s.data, pixels) || s.writes != 11 || s.sizes[10] != 24 {
		t.Fatal(s)
	}
	if err = c.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = c.Abort(); err != nil || !slices.Equal([]int{s.begin, s.commits, s.aborts}, []int{1, 1, 1}) {
		t.Fatal(s, err)
	}
}

func TestChunkedSinkZeroValueRejects(t *testing.T) {
	for _, c := range []*ChunkedSink{nil, {}, {sink: &sink{}}, {sink: &sink{}, limit: 4097}} {
		for _, operation := range []func() error{c.Begin, c.Commit, c.Abort, func() error { return c.Write(0, 0, []byte{1}) }} {
			if err := operation(); err != ErrConfig {
				t.Fatal(err)
			}
		}
	}
}

func TestChunkedSinkStopsOnFailureAndBounds(t *testing.T) {
	for _, limit := range []int{0, -1, 4097} {
		if _, err := NewChunkedSink(&sink{}, limit); err != ErrConfig {
			t.Fatal(limit, err)
		}
	}
	if _, err := NewChunkedSink(nil, 100); err != ErrConfig {
		t.Fatal(err)
	}
	s := &chunkSink{failAt: 3}
	c, _ := NewChunkedSink(s, 100)
	if err := c.Write(0, 0, make([]byte, 1024)); !errors.Is(err, ErrState) || s.writes != 3 || len(s.data) != 200 {
		t.Fatal(s, err)
	}
	for _, bad := range []struct {
		offset uint32
		data   []byte
	}{{0, nil}, {0, make([]byte, 4097)}, {^uint32(0), []byte{1}}} {
		if err := c.Write(0, bad.offset, bad.data); err != ErrChunk || s.writes != 3 {
			t.Fatal(s, err)
		}
	}
}

func TestChunkedSinkSteadyStateAllocations(t *testing.T) {
	s := &sink{}
	c, _ := NewChunkedSink(s, 100)
	var input [1000]byte
	allocations := testing.AllocsPerRun(100, func() {
		if err := c.Write(0, 0, input[:]); err != nil {
			panic(err)
		}
	})
	if allocations != 0 {
		t.Fatal(allocations)
	}
}
