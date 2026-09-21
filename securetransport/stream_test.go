package securetransport

import (
	"bytes"
	"io"
	"net"
	"testing"
)

func TestRecordStreamRoundTripAndFragmentedReads(t *testing.T) {
	hostSession, deviceSession := sessionPair(t)
	hostConn, deviceConn := net.Pipe()
	defer func() { _ = hostConn.Close() }()
	defer func() { _ = deviceConn.Close() }()
	host := NewRecordStream(hostConn, hostSession)
	device := NewRecordStream(deviceConn, deviceSession)
	want := bytes.Repeat([]byte{0x5a}, MaxPlaintext)
	done := make(chan error, 1)
	go func() {
		_, err := host.Write(want)
		done <- err
	}()
	got := make([]byte, len(want))
	for offset := 0; offset < len(got); {
		end := min(offset+7, len(got))
		n, err := device.Read(got[offset:end])
		if err != nil {
			t.Fatal(err)
		}
		offset += n
	}
	if err := <-done; err != nil || !bytes.Equal(got, want) {
		t.Fatalf("equal=%t write error=%v", bytes.Equal(got, want), err)
	}
}

func TestMutualHandshakeProducesUsableStreams(t *testing.T) {
	hostConn, deviceConn := net.Pipe()
	defer func() { _ = hostConn.Close() }()
	defer func() { _ = deviceConn.Close() }()
	serverResult := make(chan *RecordStream, 1)
	serverError := make(chan error, 1)
	go func() {
		stream, err := ServerHandshake(deviceConn, testKey, testDevice, 4, 2)
		serverResult <- stream
		serverError <- err
	}()
	host, err := HostHandshake(hostConn, testKey, testDevice, testNonce)
	if err != nil {
		t.Fatal(err)
	}
	device := <-serverResult
	if err := <-serverError; err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := device.Write([]byte("hello"))
		done <- err
	}()
	got := make([]byte, 5)
	if _, err := io.ReadFull(host, got); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil || string(got) != "hello" {
		t.Fatalf("got=%q write error=%v", got, err)
	}
}

func TestRecordStreamRejectsEmptyAndOversizeWrites(t *testing.T) {
	host, _ := sessionPair(t)
	stream := NewRecordStream(&bytes.Buffer{}, host)
	if _, err := stream.Write(nil); err == nil {
		t.Fatal("empty record accepted")
	}
	if _, err := stream.Write(make([]byte, MaxPlaintext+1)); err == nil {
		t.Fatal("oversize record accepted")
	}
}
