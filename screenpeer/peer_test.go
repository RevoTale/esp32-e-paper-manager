package screenpeer

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/network"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

var testKey = securetransport.Key{3}
var testID = securetransport.DeviceID{4}

func TestAuthenticatedOutboundDeviceCarriesBothDirections(t *testing.T) {
	pico, manager := net.Pipe()
	t.Cleanup(func() { _ = pico.Close(); _ = manager.Close() })
	lifetime, _ := network.NewLifetime([8]byte{1}, 9)
	result := make(chan error, 1)
	go func() {
		stream, err := Device(pico, testKey, testID, lifetime, time.Now())
		if err == nil {
			err = echo(stream)
		}
		result <- err
	}()
	stream, id, err := Manager(manager, testLookup, time.Now())
	if err != nil || id != testID {
		t.Fatal(id, err)
	}
	if _, err = stream.Write([]byte("EPS2 request")); err != nil {
		t.Fatal(err)
	}
	var response [12]byte
	if _, err = io.ReadFull(stream, response[:]); err != nil || string(response[:]) != "EPS2 request" {
		t.Fatal(response, err)
	}
	if err = <-result; err != nil {
		t.Fatal(err)
	}
}

func echo(stream io.ReadWriter) error {
	var record [12]byte
	if _, err := io.ReadFull(stream, record[:]); err != nil {
		return err
	}
	_, err := stream.Write(record[:])
	return err
}

func testLookup(id securetransport.DeviceID) (securetransport.Key, error) {
	if id != testID {
		return securetransport.Key{}, securetransport.ErrAuthentication
	}
	return testKey, nil
}

type fakeSocket struct {
	reader   io.Reader
	writes   bytes.Buffer
	deadline []time.Time
	closed   bool
	err      error
}

func (s *fakeSocket) Read(p []byte) (int, error)  { return s.reader.Read(p) }
func (s *fakeSocket) Write(p []byte) (int, error) { return s.writes.Write(p) }
func (s *fakeSocket) Close() error                { s.closed = true; return nil }
func (s *fakeSocket) SetDeadline(t time.Time) error {
	s.deadline = append(s.deadline, t)
	return s.err
}

func TestFailedDeviceAttemptConsumesCounterBeforePreface(t *testing.T) {
	lifetime, _ := network.NewLifetime([8]byte{1}, 7)
	socket := &fakeSocket{reader: bytes.NewReader(nil)}
	if _, err := Device(socket, testKey, testID, lifetime, time.Now()); err == nil || !socket.closed {
		t.Fatal(err, socket.closed)
	}
	if got := socket.writes.Bytes(); len(got) != 20+securetransport.ChallengeSize || string(got[:4]) != "EPN2" {
		t.Fatalf("preface/challenge: %x", got)
	}
	epoch, next, err := lifetime.NextSession()
	if epoch != 7 || next != 2 || err != nil {
		t.Fatal(epoch, next, err)
	}
}
