package securetransport

import (
	"bytes"
	"errors"
	"testing"
)

var (
	testKey = Key{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
		16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	testDevice = DeviceID{'p', 'i', 'c', 'o', '-', 'e', 'p', 'a', 'p', 'e', 'r', '-', '0', '0', '0', '1'}
	testNonce  = ClientNonce{31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16,
		15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
)

func TestHandshakeAndBothDirections(t *testing.T) {
	challenge, err := NewChallenge(testKey, testDevice, 7, 3)
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
	assertRoundTrip(t, host, device, []byte("frame record"))
	assertRoundTrip(t, device, host, []byte("reply record"))
}

func TestAuthenticationRejectsTamperingAndWrongIdentity(t *testing.T) {
	challenge, _ := NewChallenge(testKey, testDevice, 7, 3)
	for i := range challenge {
		tampered := challenge
		tampered[i] ^= 1
		if _, _, err := NewHostSession(testKey, testDevice, tampered[:], testNonce); !errors.Is(err, ErrAuthentication) {
			t.Fatalf("challenge byte %d error=%v", i, err)
		}
	}
	auth, _, _ := NewHostSession(testKey, testDevice, challenge[:], testNonce)
	for i := range auth {
		tampered := auth
		tampered[i] ^= 1
		if _, err := NewDeviceSession(testKey, testDevice, challenge[:], tampered[:]); !errors.Is(err, ErrAuthentication) {
			t.Fatalf("auth byte %d error=%v", i, err)
		}
	}
	wrongID := testDevice
	wrongID[0] ^= 1
	if _, _, err := NewHostSession(testKey, wrongID, challenge[:], testNonce); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("wrong identity error=%v", err)
	}
}

func TestOpenRejectsReplayGapAndCorruptionWithoutAdvancing(t *testing.T) {
	host, device := sessionPair(t)
	first := seal(t, host, []byte("one"))
	second := seal(t, host, []byte("two"))
	dst := make([]byte, MaxPlaintext)
	if _, err := device.Open(dst, second); !errors.Is(err, ErrSequence) {
		t.Fatalf("gap error=%v", err)
	}
	corrupt := append([]byte(nil), first...)
	corrupt[len(corrupt)-1] ^= 1
	before := bytes.Repeat([]byte{0xa5}, MaxPlaintext)
	copy(dst, before)
	if _, err := device.Open(dst, corrupt); !errors.Is(err, ErrAuthentication) || !bytes.Equal(dst, before) {
		t.Fatalf("corruption error=%v mutated=%t", err, !bytes.Equal(dst, before))
	}
	if n, err := device.Open(dst, first); err != nil || string(dst[:n]) != "one" {
		t.Fatalf("valid after failure n=%d err=%v", n, err)
	}
	if _, err := device.Open(dst, first); !errors.Is(err, ErrSequence) {
		t.Fatalf("replay error=%v", err)
	}
}

func TestBoundsAndConfigurationFailClosed(t *testing.T) {
	if _, err := NewChallenge(Key{}, testDevice, 1, 1); !errors.Is(err, ErrConfig) {
		t.Fatalf("zero key error=%v", err)
	}
	if _, err := NewChallenge(testKey, DeviceID{}, 1, 1); !errors.Is(err, ErrConfig) {
		t.Fatalf("zero device error=%v", err)
	}
	if _, err := NewChallenge(testKey, testDevice, 0, 1); !errors.Is(err, ErrConfig) {
		t.Fatalf("zero epoch error=%v", err)
	}
	host, _ := sessionPair(t)
	if _, err := host.Seal(make([]byte, MaxEnvelope), make([]byte, MaxPlaintext+1)); !errors.Is(err, ErrBounds) {
		t.Fatalf("oversize error=%v", err)
	}
	if _, err := host.Seal(make([]byte, MaxEnvelope-1), make([]byte, MaxPlaintext)); !errors.Is(err, ErrBounds) {
		t.Fatalf("short output error=%v", err)
	}
}

func TestDirectionalKeysProduceDifferentCiphertext(t *testing.T) {
	host, device := sessionPair(t)
	plain := []byte("same")
	hostWire := seal(t, host, plain)
	deviceWire := seal(t, device, plain)
	if bytes.Equal(hostWire, deviceWire) {
		t.Fatal("directional ciphertexts are equal")
	}
}

func sessionPair(t *testing.T) (*Session, *Session) {
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

func seal(t *testing.T, session *Session, plain []byte) []byte {
	t.Helper()
	buffer := make([]byte, MaxEnvelope)
	n, err := session.Seal(buffer, plain)
	if err != nil {
		t.Fatal(err)
	}
	return append([]byte(nil), buffer[:n]...)
}

func assertRoundTrip(t *testing.T, sender, receiver *Session, plain []byte) {
	t.Helper()
	wire := seal(t, sender, plain)
	dst := make([]byte, MaxPlaintext)
	n, err := receiver.Open(dst, wire)
	if err != nil || !bytes.Equal(dst[:n], plain) {
		t.Fatalf("round trip n=%d err=%v", n, err)
	}
}
