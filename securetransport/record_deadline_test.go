package securetransport

import (
	"errors"
	"net"
	"testing"
	"time"
)

func TestActualSocketStallAfterFirstCiphertextByteTimesOut(t *testing.T) {
	_, device := sessionPair(t)
	sender, receiver := net.Pipe()
	t.Cleanup(func() { _ = sender.Close(); _ = receiver.Close() })
	done := make(chan error, 1)
	go func() { _, err := sender.Write([]byte{0}); done <- err }()
	stream := NewRecordStream(receiver, device)
	var output [MaxPlaintext]byte
	limits := ReadLimits{Idle: time.Second, Active: 20 * time.Millisecond, SetDeadline: receiver.SetDeadline}
	_, err := stream.ReadRecord(output[:], limits)
	var timeout net.Error
	if !errors.As(err, &timeout) || !timeout.Timeout() {
		t.Fatal(err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if stream.stream != nil {
		t.Fatal("timed out stream reusable")
	}
}
