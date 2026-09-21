package screenpeer

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/network"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

func TestOldAuthenticationCannotOpenNextConnection(t *testing.T) {
	challenge, err := securetransport.NewChallenge(testKey, testID, 7, 1)
	if err != nil {
		t.Fatal(err)
	}
	auth, _, err := securetransport.NewHostSession(testKey, testID, challenge[:], securetransport.ClientNonce{9})
	if err != nil {
		t.Fatal(err)
	}
	lifetime, _ := network.NewLifetime([8]byte{1}, 7)
	first := &fakeSocket{reader: bytes.NewReader(auth[:])}
	if _, err = Device(first, testKey, testID, lifetime, time.Now()); err != nil {
		t.Fatal(err)
	}
	replay := &fakeSocket{reader: bytes.NewReader(auth[:])}
	stream, err := Device(replay, testKey, testID, lifetime, time.Now())
	if !errors.Is(err, securetransport.ErrAuthentication) || stream != nil || !replay.closed {
		t.Fatal("old proof accepted", stream, err, replay.closed)
	}
	if bytes.Equal(first.writes.Bytes(), replay.writes.Bytes()) {
		t.Fatal("challenge reused")
	}
}

func TestWrongKeyAndTruncatedProofNeverReturnAuthenticatedStream(t *testing.T) {
	challenge, _ := securetransport.NewChallenge(testKey, testID, 7, 1)
	auth, _, _ := securetransport.NewHostSession(testKey, testID, challenge[:], securetransport.ClientNonce{9})
	for n := 0; n < len(auth); n++ {
		lifetime, _ := network.NewLifetime([8]byte{1}, 7)
		socket := &fakeSocket{reader: bytes.NewReader(auth[:n])}
		if stream, err := Device(socket, testKey, testID, lifetime, time.Now()); err == nil || stream != nil || !socket.closed {
			t.Fatal(n, err)
		}
	}
	lifetime, _ := network.NewLifetime([8]byte{1}, 7)
	socket := &fakeSocket{reader: bytes.NewReader(auth[:])}
	if _, err := Device(socket, securetransport.Key{99}, testID, lifetime, time.Now()); !errors.Is(err, securetransport.ErrAuthentication) {
		t.Fatal(err)
	}
}
