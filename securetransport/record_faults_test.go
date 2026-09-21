package securetransport

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func recordFixture(t *testing.T, plaintext []byte) (*RecordStream, *countedReader) {
	t.Helper()
	host, device := sessionPair(t)
	var envelope [MaxEnvelope]byte
	n, err := host.Seal(envelope[:], plaintext)
	if err != nil {
		t.Fatal(err)
	}
	r := &countedReader{Reader: bytes.NewReader(envelope[:n])}
	return NewRecordStream(r, device), r
}

func TestRecordConfigurationRejectsBeforeConsumingCiphertext(t *testing.T) {
	for _, limits := range []ReadLimits{{}, {Idle: time.Second, SetDeadline: func(time.Time) error { return nil }},
		{Idle: time.Second, Active: time.Minute, SetDeadline: func(time.Time) error { return nil }}} {
		s, r := recordFixture(t, []byte("hello"))
		var output [MaxPlaintext]byte
		if _, err := s.ReadRecord(output[:], limits); err == nil || r.consumed != 0 {
			t.Fatal(err, r.consumed)
		}
	}
	s, r := recordFixture(t, []byte("hello"))
	if _, err := s.ReadRecord(make([]byte, MaxPlaintext-1), testLimits()); err != ErrBounds || r.consumed != 0 {
		t.Fatal(err, r.consumed)
	}
}

func TestEmptyTamperedAndOversizeEnvelopeReleasesNoPlaintext(t *testing.T) {
	for _, kind := range []string{"empty", "tamper", "oversize"} {
		s, _ := recordFixture(t, []byte("hello"))
		if kind == "empty" {
			s, _ = recordFixture(t, nil)
		}
		ciphertext, _ := io.ReadAll(s.stream)
		if kind == "tamper" {
			ciphertext[len(ciphertext)-1] ^= 1
		}
		if kind == "oversize" {
			ciphertext[8], ciphertext[9] = 255, 255
		}
		s.stream = bytes.NewBuffer(ciphertext)
		output := bytes.Repeat([]byte{0xa5}, MaxPlaintext)
		if n, err := s.ReadRecord(output, testLimits()); n != 0 || err == nil {
			t.Fatal(kind, n, err)
		}
		if !bytes.Equal(output, bytes.Repeat([]byte{0xa5}, MaxPlaintext)) {
			t.Fatal("unauthenticated plaintext escaped")
		}
	}
}

func TestEncryptedFirstByteThenStallUsesOneActiveDeadline(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		s, r := recordFixture(t, []byte("hello"))
		calls := 0
		limits := testLimits()
		limits.SetDeadline = func(time.Time) error {
			calls++
			if calls == failAt {
				return io.ErrClosedPipe
			}
			return nil
		}
		var output [MaxPlaintext]byte
		if _, err := s.ReadRecord(output[:], limits); err != io.ErrClosedPipe {
			t.Fatal(err)
		}
		if r.consumed != failAt-1 {
			t.Fatal("unexpected bytes before failed deadline", r.consumed)
		}
	}
	s, _ := recordFixture(t, []byte("hello"))
	s.stream = &stalledCipher{}
	var output [MaxPlaintext]byte
	if _, err := s.ReadRecord(output[:], testLimits()); err != io.ErrNoProgress {
		t.Fatal(err)
	}
}

type stalledCipher struct{ first bool }

func (s *stalledCipher) Read(p []byte) (int, error) {
	if !s.first {
		s.first = true
		p[0] = 0
		return 1, nil
	}
	return 0, io.ErrNoProgress
}
func (*stalledCipher) Write(p []byte) (int, error) { return len(p), nil }
