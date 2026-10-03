package interop

import (
	"errors"
	"io"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

// Consume a genuine complete native reply, then simulate its loss. Reconnect
// retains the receiver process/boot; native SPI counts catch accidental replay.
type lostCommitStream struct {
	io.ReadWriter
	enabled, armed, dropped bool
}

func (s *lostCommitStream) Write(p []byte) (int, error) {
	r, err := screenwire.Decode(p)
	if err != nil {
		return 0, err
	}
	if s.enabled && !s.dropped && r.Kind == screenwire.Commit {
		s.armed = true
	}
	return s.ReadWriter.Write(p)
}

func (s *lostCommitStream) Read(p []byte) (int, error) {
	if !s.armed {
		return s.ReadWriter.Read(p)
	}
	var b [screenwire.MaxRecord]byte
	if _, err := io.ReadFull(s.ReadWriter, b[:screenwire.HeaderSize]); err != nil {
		return 0, err
	}
	n, err := screenwire.Size(b[:screenwire.HeaderSize])
	if err != nil {
		return 0, err
	}
	if _, err := io.ReadFull(s.ReadWriter, b[screenwire.HeaderSize:n]); err != nil {
		return 0, err
	}
	if _, err := screenwire.Decode(b[:n]); err != nil {
		return 0, err
	}
	s.armed, s.dropped = false, true
	return 0, io.ErrUnexpectedEOF
}

func reconcileNativeRegion(t *testing.T, client *screenclient.Client, stream *lostCommitStream, sendErr error) {
	t.Helper()
	if !errors.Is(sendErr, io.ErrUnexpectedEOF) || !client.Pending() || client.Baseline() != [32]byte{} {
		t.Fatal("lost completion was not retained", sendErr)
	}
	if _, err := client.Connect(stream); err != nil {
		t.Fatal(err)
	}
	result, err := client.Reconcile()
	if err != nil || !result.Confirmed || !stream.dropped {
		t.Fatal(result, err)
	}
}
