package provisionclient

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

type loopback struct {
	request bytes.Buffer
	reply   []byte
	step    int
}

func TestClientRejectsNilAndTransportFailures(t *testing.T) {
	if _, err := New(nil); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("nil=%v", err)
	}
	client := &Client{}
	if _, err := client.Execute(provision.Request{Operation: provision.OperationInspect}); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("empty client=%v", err)
	}
	broken, _ := New(failingStream{})
	if _, err := broken.Execute(provision.Request{Operation: provision.OperationInspect}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("broken=%v", err)
	}
}

type failingStream struct{}

func (failingStream) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func (failingStream) Read([]byte) (int, error)  { return 0, io.ErrClosedPipe }

func (stream *loopback) Write(value []byte) (int, error) {
	limit := min(len(value), 7)
	return stream.request.Write(value[:limit])
}

func (stream *loopback) Read(value []byte) (int, error) {
	if stream.step >= len(stream.reply) {
		return 0, nil
	}
	limit := min(len(value), 11, len(stream.reply)-stream.step)
	copy(value, stream.reply[stream.step:stream.step+limit])
	stream.step += limit
	return limit, nil
}

func TestClientHandlesShortReadsAndWrites(t *testing.T) {
	reply := make([]byte, provision.ResponseSize)
	want := provision.Response{Operation: provision.OperationInspect, State: provision.StateBlank}
	if err := provision.EncodeResponse(reply, want); err != nil {
		t.Fatal(err)
	}
	stream := &loopback{reply: reply}
	client, err := New(stream)
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.Execute(provision.Request{Operation: provision.OperationInspect})
	if err != nil || got != want || stream.request.Len() != provision.RequestSize {
		t.Fatalf("got=%+v bytes=%d err=%v", got, stream.request.Len(), err)
	}
}

type responseStream struct {
	reply    []byte
	count    int
	badWrite bool
}

func (stream *responseStream) Write(value []byte) (int, error) {
	if stream.badWrite {
		return len(value) + 1, nil
	}
	return len(value), nil
}
func (stream *responseStream) Read(value []byte) (int, error) {
	if stream.count > 0 {
		return 0, io.EOF
	}
	stream.count++
	return copy(value, stream.reply), nil
}

func TestClientRejectsInvalidRequestResponseAndWriteCount(t *testing.T) {
	client, _ := New(&responseStream{})
	if _, err := client.Execute(provision.Request{}); err == nil {
		t.Fatal("invalid request accepted")
	}
	client, _ = New(&responseStream{reply: make([]byte, provision.ResponseSize)})
	if _, err := client.Execute(provision.Request{Operation: provision.OperationInspect}); err == nil {
		t.Fatal("invalid response accepted")
	}
	client, _ = New(&responseStream{badWrite: true})
	if _, err := client.Execute(provision.Request{Operation: provision.OperationInspect}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("oversized write=%v", err)
	}
}
