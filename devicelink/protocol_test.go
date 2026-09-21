package devicelink

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/manager"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
	"github.com/RevoTale/esp32-e-paper-manager/update"
)

func TestRejectsInvalidDeviceConfigAndCounters(t *testing.T) {
	if _, err := ConnectDevice(nil, DeviceConfig{}, bytes.NewReader(nil)); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("config=%v", err)
	}
	if _, _, err := randomCounters(bytes.NewReader(make([]byte, 16))); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("zero counters=%v", err)
	}
	if _, _, err := randomCounters(bytes.NewReader([]byte{1})); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("short counters=%v", err)
	}
	if err := (*DeviceSession)(nil).SendResult(update.Result{}); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("nil result=%v", err)
	}
	if _, _, err := (*DeviceSession)(nil).Receive(); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("nil receive=%v", err)
	}
	if err := ServeManager(nil, ManagerConfig{}); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("nil manager=%v", err)
	}
}

func TestDeviceSessionRejectsInvalidReportsAndMessages(t *testing.T) {
	device, server := sessionPair(t)
	if device.Generation() != 0 || (*DeviceSession)(nil).Generation() != 0 {
		t.Fatal("unexpected initial generation")
	}
	if err := device.Report(&update.Result{}, 0); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("invalid report=%v", err)
	}
	if err := (*DeviceSession)(nil).Report(nil, 0); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("nil report=%v", err)
	}
	go func() { _ = writeMessage(server, messageUpdate, 1, []byte{1}) }()
	if _, _, err := device.Receive(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("short update=%v", err)
	}
	device, server = sessionPair(t)
	go func() { _ = writeMessage(server, messageUpdate, 2, make([]byte, update.HeaderSize+update.TrailerSize)) }()
	if _, _, err := device.Receive(); err == nil {
		t.Fatal("invalid encoded update accepted")
	}
}

func TestDeviceSessionIdleAndResultValidation(t *testing.T) {
	device, server := sessionPair(t)
	go func() { _ = writeMessage(server, messageIdle, 0, nil) }()
	if _, available, err := device.Receive(); err != nil || available {
		t.Fatalf("available=%v err=%v", available, err)
	}
	if err := device.SendResult(update.Result{}); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("invalid result=%v", err)
	}
}

func sessionPair(t *testing.T) (*DeviceSession, *securetransport.RecordStream) {
	t.Helper()
	serverSide, deviceSide := net.Pipe()
	key, id := securetransport.Key{2}, securetransport.DeviceID{1}
	type outcome struct {
		stream *securetransport.RecordStream
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		var received securetransport.DeviceID
		_, err := io.ReadFull(serverSide, received[:])
		if err == nil {
			var stream *securetransport.RecordStream
			stream, err = securetransport.ServerHandshake(serverSide, key, received, 1, 2)
			done <- outcome{stream: stream, err: err}
			return
		}
		done <- outcome{err: err}
	}()
	device, err := ConnectDevice(deviceSide, DeviceConfig{ID: id, Key: key}, bytes.NewReader(bytes.Repeat([]byte{3}, 32)))
	server := <-done
	if err != nil || server.err != nil {
		t.Fatalf("device=%v server=%v", err, server.err)
	}
	return device, server.stream
}

func TestProtocolHeaderValidation(t *testing.T) {
	for _, wire := range [][]byte{
		make([]byte, headerSize),
		append([]byte("EPD2\x09"), make([]byte, headerSize-5)...),
		[]byte("short"),
	} {
		if _, err := readHeader(bytes.NewReader(wire)); err == nil {
			t.Fatalf("accepted %x", wire)
		}
	}
	if err := writeMessage(nil, messageIdle, 0, nil); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("nil stream=%v", err)
	}
}

func TestValidHeaderRoundTrip(t *testing.T) {
	var wire [headerSize]byte
	copy(wire[:4], "EPD2")
	wire[4] = byte(messageIdle)
	header, err := readHeader(bytes.NewReader(wire[:]))
	if err != nil || header.typeID != messageIdle || header.length != 0 {
		t.Fatalf("header=%+v err=%v", header, err)
	}
}

func TestWriteFullAndChunkedRejectStalls(t *testing.T) {
	stall := stalledWriter{}
	if err := writeFull(stall, []byte{1}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("write full=%v", err)
	}
	if err := writeChunked(stall, []byte{1}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("write chunked=%v", err)
	}
}

type stalledWriter struct{}

func (stalledWriter) Write([]byte) (int, error) { return 0, nil }

type oversizedWriter struct{}

func (oversizedWriter) Write(value []byte) (int, error) { return len(value) + 1, nil }

func TestWriteFullRejectsOversizedCount(t *testing.T) {
	if err := writeFull(oversizedWriter{}, []byte{1}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("error=%v", err)
	}
}

func TestManagerRejectsInvalidRetainedAndImmediateMessages(t *testing.T) {
	config := ManagerConfig{Store: manager.NewStore(), Now: time.Now}
	id := securetransport.DeviceID{1}
	device, server := sessionPair(t)
	go func() { _ = writeMessage(device.stream, messageUpdate, 1, []byte{1}) }()
	if err := receiveRetained(server, id, config); !errors.Is(err, ErrProtocol) {
		t.Fatalf("retained type=%v", err)
	}
	device, server = sessionPair(t)
	go func() { _ = writeMessage(device.stream, messageResult, 1, make([]byte, update.ResultEncodedSize)) }()
	if err := receiveRetained(server, id, config); err == nil {
		t.Fatal("invalid retained result accepted")
	}
	device, server = sessionPair(t)
	go func() { _ = writeMessage(device.stream, messageIdle, 1, nil) }()
	if err := receiveResult(server, id, 1, config); !errors.Is(err, ErrProtocol) {
		t.Fatalf("immediate type=%v", err)
	}
}
