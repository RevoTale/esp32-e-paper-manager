package screenpeer

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

func validInput() []byte {
	challenge, _ := securetransport.NewChallenge(testKey, testID, 7, 1)
	p := append([]byte("EPN2"), testID[:]...)
	return append(p, challenge[:]...)
}

func TestManagerRejectsEveryTruncationBeforeAuthentication(t *testing.T) {
	input := validInput()
	for n := 0; n < len(input); n++ {
		socket := &fakeSocket{reader: bytes.NewReader(input[:n])}
		stream, id, err := Manager(socket, testLookup, time.Now())
		if err == nil || stream != nil || id != (securetransport.DeviceID{}) || !socket.closed {
			t.Fatal(n, id, err)
		}
		if socket.writes.Len() != 0 {
			t.Fatal("unverified authentication emitted", n)
		}
	}
}

func TestManagerRejectsInvalidIdentityKeyAndRandomSource(t *testing.T) {
	for _, change := range []func([]byte){
		func(p []byte) { p[3] = '1' },
		func(p []byte) { clear(p[4:20]) },
		func(p []byte) { p[4]++ },
		func(p []byte) { p[len(p)-1]++ },
	} {
		input := validInput()
		change(input)
		socket := &fakeSocket{reader: bytes.NewReader(input)}
		if _, _, err := Manager(socket, testLookup, time.Now()); err == nil || socket.writes.Len() != 0 {
			t.Fatal(err)
		}
	}
	for _, random := range []io.Reader{nil, bytes.NewReader(nil), bytes.NewReader(make([]byte, 32))} {
		socket := &fakeSocket{reader: bytes.NewReader(validInput())}
		if _, _, err := accept(socket, testLookup, time.Now(), random); err == nil || !socket.closed {
			t.Fatal(err)
		}
	}
}

func TestHandshakeDeadlineAndSuccessfulManagerProof(t *testing.T) {
	socket := &fakeSocket{reader: bytes.NewReader(validInput())}
	now := time.Unix(7, 0)
	stream, id, err := Manager(socket, testLookup, now)
	if err != nil || stream == nil || id != testID || socket.closed {
		t.Fatal(id, err)
	}
	if socket.writes.Len() != securetransport.AuthSize {
		t.Fatal(socket.writes.Len())
	}
	if len(socket.deadline) != 2 || socket.deadline[0] != now.Add(HandshakeTimeout) || !socket.deadline[1].IsZero() {
		t.Fatal(socket.deadline)
	}
}
