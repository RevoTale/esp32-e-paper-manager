package interop

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

func device(t *testing.T, corruptAuth bool) (io.WriteCloser, io.ReadCloser, *securetransport.Session, func()) {
	t.Helper()
	peer := startNativePeer(t)
	auth, session := authenticateChallenge(t, peer.output)
	if corruptAuth {
		auth[40] ^= 1
	}
	if _, err := peer.input.Write(auth[:]); err != nil {
		t.Fatal(err)
	}
	var ok [1]byte
	if _, err := io.ReadFull(peer.output, ok[:]); err != nil {
		t.Fatal(err)
	}
	if (ok[0] == 1) == corruptAuth {
		t.Fatalf("wrong authentication decision: %d", ok[0])
	}
	return peer.input, peer.output, session, peer.finish
}

func authenticateChallenge(t *testing.T, output io.Reader) (securetransport.Auth, *securetransport.Session) {
	t.Helper()
	var key securetransport.Key
	for i := range key {
		key[i] = byte(i)
	}
	var id securetransport.DeviceID
	copy(id[:], "pico-epaper-0001")
	var challenge securetransport.Challenge
	if _, err := io.ReadFull(output, challenge[:]); err != nil {
		t.Fatal(err)
	}
	expected, err := securetransport.NewChallenge(key, id, 7, 3)
	if err != nil || challenge != expected {
		t.Fatalf("C challenge differs from Go: %v", err)
	}
	var nonce securetransport.ClientNonce
	for i := range nonce {
		nonce[i] = byte(31 - i)
	}
	auth, session, err := securetransport.NewHostSession(key, id, challenge[:], nonce)
	if err != nil {
		t.Fatal(err)
	}
	return auth, session
}

func TestCDeviceAuthenticatesAndExchangesWithGo(t *testing.T) {
	input, output, session, finish := device(t, false)
	defer finish()
	for _, size := range []int{1, 32, 512, 1056} {
		plain := bytes.Repeat([]byte{byte(size)}, size)
		wire := make([]byte, securetransport.MaxEnvelope)
		n, err := session.Seal(wire, plain)
		if err != nil {
			t.Fatal(err)
		}
		var length [2]byte
		binary.BigEndian.PutUint16(length[:], uint16(n))
		if _, err = input.Write(append(length[:], wire[:n]...)); err != nil {
			t.Fatal(err)
		}
		reply := readAcceptedEnvelope(t, output, wire)
		actual := make([]byte, len(plain))
		count, err := session.Open(actual, reply)
		if err != nil || count != len(plain) || !bytes.Equal(actual, plain) {
			t.Fatalf("roundtrip: %v", err)
		}
	}
}

func readAcceptedEnvelope(t *testing.T, output io.Reader, wire []byte) []byte {
	t.Helper()
	var accepted [1]byte
	if _, err := io.ReadFull(output, accepted[:]); err != nil || accepted[0] != 1 {
		t.Fatalf("open: %v %v", accepted, err)
	}
	var length [2]byte
	if _, err := io.ReadFull(output, length[:]); err != nil {
		t.Fatal(err)
	}
	n := int(binary.BigEndian.Uint16(length[:]))
	if n > len(wire) {
		t.Fatal("oversized C reply")
	}
	if _, err := io.ReadFull(output, wire[:n]); err != nil {
		t.Fatal(err)
	}
	return wire[:n]
}

func TestCDeviceRejectsWrongProof(t *testing.T) {
	_, _, _, finish := device(t, true)
	finish()
}
