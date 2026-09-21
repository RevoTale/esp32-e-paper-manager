package securetransport

import (
	"bytes"
	"io"
	"testing"
	"time"
)

type countedReader struct {
	*bytes.Reader
	consumed int
}

func (r *countedReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p[:min(len(p), 3)])
	r.consumed += n
	return n, err
}
func (*countedReader) Write(p []byte) (int, error) { return len(p), nil }

func TestRecordDeadlineStartsAfterFirstEncryptedByteNotAfterDecryption(t *testing.T) {
	host, device := sessionPair(t)
	var envelope [MaxEnvelope]byte
	n, _ := host.Seal(envelope[:], []byte("fixture"))
	reader := &countedReader{Reader: bytes.NewReader(envelope[:n])}
	stream := NewRecordStream(reader, device)
	var positions []int
	limits := ReadLimits{Idle: 15 * time.Minute, Active: 20 * time.Second,
		SetDeadline: func(time.Time) error { positions = append(positions, reader.consumed); return nil }}
	var output [MaxPlaintext]byte
	n, err := stream.ReadRecord(output[:], limits)
	if err != nil || string(output[:n]) != "fixture" {
		t.Fatal(n, err)
	}
	if len(positions) != 2 || positions[0] != 0 || positions[1] != 1 {
		t.Fatal("deadline starts too late", positions)
	}
}

func TestRecordReadRejectsTruncationAndPoisonsTransport(t *testing.T) {
	for size := 0; size < HeaderSize+7+TagSize; size++ {
		host, device := sessionPair(t)
		var envelope [MaxEnvelope]byte
		_, _ = host.Seal(envelope[:], []byte("fixture"))
		s := NewRecordStream(&countedReader{Reader: bytes.NewReader(envelope[:size])}, device)
		var output [MaxPlaintext]byte
		if _, err := s.ReadRecord(output[:], testLimits()); err == nil {
			t.Fatal("truncated envelope", size)
		}
		if _, err := s.ReadRecord(output[:], testLimits()); err != ErrConfig {
			t.Fatal("failed transport reused", err)
		}
	}
}

func TestRecordAPIRejectsMixingWithPartialStreamReads(t *testing.T) {
	host, device := sessionPair(t)
	var envelope [MaxEnvelope]byte
	n, _ := host.Seal(envelope[:], []byte("fixture"))
	s := NewRecordStream(bytes.NewBuffer(envelope[:n]), device)
	var first [1]byte
	if _, err := io.ReadFull(s, first[:]); err != nil {
		t.Fatal(err)
	}
	var output [MaxPlaintext]byte
	if _, err := s.ReadRecord(output[:], testLimits()); err != ErrConfig {
		t.Fatal("mixed partial read", err)
	}
}

func testLimits() ReadLimits {
	return ReadLimits{Idle: time.Minute, Active: time.Second, SetDeadline: func(time.Time) error { return nil }}
}
