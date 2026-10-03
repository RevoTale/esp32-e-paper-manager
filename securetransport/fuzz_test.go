package securetransport

import (
	"bytes"
	"testing"
)

func FuzzOpenRejectsHostileEnvelopesWithoutMutation(f *testing.F) {
	host, _ := fuzzSessionPair(f)
	validBuffer := make([]byte, MaxEnvelope)
	validLength, err := host.Seal(validBuffer, []byte("valid"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add([]byte{})
	f.Add([]byte("short"))
	f.Add(append([]byte(nil), validBuffer[:validLength]...))
	f.Fuzz(func(t *testing.T, envelope []byte) {
		_, device := fuzzSessionPair(t)
		dst := bytes.Repeat([]byte{0xa5}, MaxPlaintext)
		before := append([]byte(nil), dst...)
		_, openErr := device.Open(dst, envelope)
		if openErr != nil && !bytes.Equal(dst, before) {
			t.Fatal("failed Open mutated destination")
		}
	})
}

func FuzzHandshakeRejectsHostileBytes(f *testing.F) {
	challenge, err := NewChallenge(testKey, testDevice, 1, 1)
	if err != nil {
		f.Fatal(err)
	}
	auth, _, err := NewHostSession(testKey, testDevice, challenge[:], testNonce)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(append([]byte(nil), challenge[:]...), append([]byte(nil), auth[:]...))
	f.Add([]byte{}, []byte{})
	f.Fuzz(func(t *testing.T, challengeWire, authWire []byte) {
		_, _ = NewDeviceSession(testKey, testDevice, challengeWire, authWire)
	})
}

type fataler interface {
	Helper()
	Fatal(args ...any)
}

func fuzzSessionPair(t fataler) (*Session, *Session) {
	t.Helper()
	challenge, err := NewChallenge(testKey, testDevice, 9, 4)
	if err != nil {
		t.Fatal(err)
	}
	auth, host, err := NewHostSession(testKey, testDevice, challenge[:], testNonce)
	if err != nil {
		t.Fatal(err)
	}
	device, err := NewDeviceSession(testKey, testDevice, challenge[:], auth[:])
	if err != nil {
		t.Fatal(err)
	}
	return host, device
}
