package screenpeer

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/network"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

type brokenSocket struct {
	fakeSocket
	readCount, writeCount int
	readErr, writeErr     error
}

func (s *brokenSocket) Read([]byte) (int, error)  { return s.readCount, s.readErr }
func (s *brokenSocket) Write([]byte) (int, error) { return s.writeCount, s.writeErr }

func TestBrokenAdaptersCannotSpinOrEmitAuthenticatedStream(t *testing.T) {
	for _, s := range []*brokenSocket{
		{}, {readCount: -1}, {readCount: 9999}, {readErr: io.ErrClosedPipe},
	} {
		if _, _, err := Manager(s, testLookup, time.Now()); err == nil || !s.closed {
			t.Fatal(err)
		}
	}
	for _, s := range []*brokenSocket{
		{}, {writeCount: -1}, {writeCount: 9999}, {writeErr: io.ErrClosedPipe},
	} {
		lifetime, _ := network.NewLifetime([8]byte{1}, 7)
		if _, err := Device(s, testKey, testID, lifetime, time.Now()); err == nil || !s.closed {
			t.Fatal(err)
		}
	}
}

func TestMissingConfigurationAndDeadlineFailuresCloseConnections(t *testing.T) {
	if _, err := Device(nil, testKey, testID, nil, time.Now()); err == nil {
		t.Fatal("nil device socket")
	}
	if _, _, err := Manager(nil, testLookup, time.Now()); err == nil {
		t.Fatal("nil manager socket")
	}
	for _, key := range []securetransport.Key{{}, testKey} {
		s := &fakeSocket{reader: bytes.NewReader(nil)}
		if _, err := Device(s, key, testID, nil, time.Now()); err == nil || !s.closed {
			t.Fatal(err)
		}
	}
	for _, lookup := range []Lookup{nil, func(securetransport.DeviceID) (securetransport.Key, error) { return securetransport.Key{}, nil }} {
		s := &fakeSocket{reader: bytes.NewReader(validInput())}
		if _, _, err := Manager(s, lookup, time.Now()); err == nil || !s.closed {
			t.Fatal(err)
		}
	}
	assertDeadlineFailure(t)
}

func assertDeadlineFailure(t *testing.T) {
	t.Helper()
	lifetime, _ := network.NewLifetime([8]byte{1}, 7)
	s := &fakeSocket{reader: bytes.NewReader(nil), err: io.ErrClosedPipe}
	if _, err := Device(s, testKey, testID, lifetime, time.Now()); err == nil || s.writes.Len() != 0 || !s.closed {
		t.Fatal(err)
	}
	s = &fakeSocket{reader: bytes.NewReader(validInput()), err: io.ErrClosedPipe}
	if _, _, err := Manager(s, testLookup, time.Now()); err == nil || !s.closed {
		t.Fatal(err)
	}
}

type clearDeadlineFailure struct{ fakeSocket }

func (s *clearDeadlineFailure) SetDeadline(at time.Time) error {
	if at.IsZero() {
		return io.ErrClosedPipe
	}
	return nil
}

func TestClearingDeadlineFailureDoesNotReturnLiveStream(t *testing.T) {
	s := &clearDeadlineFailure{fakeSocket{reader: bytes.NewReader(validInput())}}
	stream, id, err := Manager(s, testLookup, time.Now())
	if err == nil || stream != nil || id != (securetransport.DeviceID{}) || !s.closed {
		t.Fatal(id, err)
	}
}
