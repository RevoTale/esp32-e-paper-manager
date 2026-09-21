package screenpeer

import (
	"bytes"
	"io"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

func replyFixture(t *testing.T) []byte {
	t.Helper()
	var body [screenwire.StatusSize]byte
	s := screenwire.Status{Operation: screenwire.Abort, Generation: 1, Boot: [16]byte{1}}
	if err := screenwire.EncodeStatus(body[:], s); err != nil {
		t.Fatal(err)
	}
	var wire [screenwire.MaxRecord]byte
	n, err := screenwire.Encode(wire[:], screenwire.Record{Kind: screenwire.Reply, Epoch: 1, Payload: body[:]})
	if err != nil {
		t.Fatal(err)
	}
	return append([]byte(nil), wire[:n]...)
}

func hostFixture(t *testing.T, plaintexts ...[]byte) (*HostScreen, *fakeSocket) {
	t.Helper()
	challenge, _ := securetransport.NewChallenge(testKey, testID, 7, 1)
	auth, host, _ := securetransport.NewHostSession(testKey, testID, challenge[:], securetransport.ClientNonce{9})
	device, _ := securetransport.NewDeviceSession(testKey, testID, challenge[:], auth[:])
	var ciphertext bytes.Buffer
	for _, p := range plaintexts {
		var envelope [securetransport.MaxEnvelope]byte
		n, err := device.Seal(envelope[:], p)
		if err != nil {
			t.Fatal(err)
		}
		ciphertext.Write(envelope[:n])
	}
	socket := &fakeSocket{reader: bytes.NewReader(ciphertext.Bytes())}
	stream, err := NewHostScreen(socket, securetransport.NewRecordStream(socket, host))
	if err != nil {
		t.Fatal(err)
	}
	return stream, socket
}

func TestHostReplyEnvelopeIsValidatedBeforeFragmentedClientReads(t *testing.T) {
	want := replyFixture(t)
	stream, socket := hostFixture(t, want)
	var output bytes.Buffer
	for offset := 0; offset < len(want); {
		var part [7]byte
		n, err := stream.Read(part[:])
		if err != nil {
			t.Fatal(err)
		}
		output.Write(part[:n])
		offset += n
	}
	if !bytes.Equal(output.Bytes(), want) || socket.closed {
		t.Fatal("fragmented reply mismatch")
	}
}

func TestSplitAndBundledRepliesExposeNoPlaintext(t *testing.T) {
	reply := replyFixture(t)
	for _, packets := range [][][]byte{{reply[:20], reply[20:]}, {append(append([]byte(nil), reply...), reply...)}, {[]byte("EPCQ")}} {
		s, socket := hostFixture(t, packets...)
		var output [7]byte
		if n, err := s.Read(output[:]); n != 0 || err == nil || !socket.closed {
			t.Fatal(n, err)
		}
		if _, err := s.Read(output[:]); err == nil {
			t.Fatal("poisoned read reused")
		}
		if _, err := s.Write(reply); err == nil {
			t.Fatal("poisoned writer reused")
		}
	}
}

func TestHostNeverWritesBeforeConsumingPreviousReply(t *testing.T) {
	s, socket := hostFixture(t, replyFixture(t))
	var one [1]byte
	if _, err := io.ReadFull(s, one[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Write(replyFixture(t)); err == nil || !socket.closed {
		t.Fatal("interleaved record admitted")
	}
}
