package screenpeer

import (
	"io"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

func TestHostWritesExactlyOneRequestEnvelopeAndRejectsMisuse(t *testing.T) {
	s, socket := hostFixture(t)
	var request [screenwire.HeaderSize]byte
	_, _ = screenwire.Encode(request[:], screenwire.Record{Kind: screenwire.Hello})
	if n, err := s.Write(request[:]); err != nil || n != len(request) {
		t.Fatal(n, err)
	}
	if socket.writes.Len() != 64 {
		t.Fatal("wrong envelope bytes", socket.writes.Len())
	}
	if n, err := s.Read(nil); n != 0 || err != nil {
		t.Fatal(n, err)
	}
	if _, err := s.Write(replyFixture(t)); err == nil || !socket.closed {
		t.Fatal("reply written as request")
	}
	if _, err := NewHostScreen(nil, nil); err == nil {
		t.Fatal("missing socket")
	}
}

func TestHostDeadlineAndRecordFailurePoisonWriter(t *testing.T) {
	var request [screenwire.HeaderSize]byte
	_, _ = screenwire.Encode(request[:], screenwire.Record{Kind: screenwire.Hello})
	for _, deadlineError := range []bool{true, false} {
		s, socket := hostFixture(t)
		if deadlineError {
			socket.err = io.ErrClosedPipe
		} else {
			s.records = securetransport.NewRecordStream(socket, nil)
		}
		if _, err := s.Write(request[:]); err == nil || !socket.closed {
			t.Fatal(err)
		}
		if _, err := s.Write(request[:]); err == nil {
			t.Fatal("failed stream reused")
		}
	}
}
