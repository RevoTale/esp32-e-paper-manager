package streamwire

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestSemanticRejections(t *testing.T) {
	l, _ := linkFixture(t)
	for _, r := range []Record{{Kind: Hello}, {Kind: Hello, Epoch: 7, ID: 1}, {Kind: Begin, Epoch: 7, ID: 1}} {
		if reply := exchange(t, l, r, 0); reply.Payload[0] == 0 {
			t.Fatal("invalid accepted", r)
		}
	}
	exchange(t, l, Record{Kind: Hello, Epoch: 7}, 0)
	for _, r := range []Record{{Kind: Hello, Epoch: 8}, {Kind: Begin, Epoch: 7, ID: 1}, {Kind: Commit, Epoch: 7, ID: 9}} {
		if reply := exchange(t, l, r, 0); reply.Payload[0] == 0 {
			t.Fatal("invalid accepted", r)
		}
	}
	d := sha256.Sum256([]byte{128, 1})
	exchange(t, l, Record{Kind: Begin, Epoch: 7, ID: 1, Payload: d[:]}, 0)
	if err := l.Tick(time.Second); err == nil {
		t.Fatal("timeout")
	}
	if reply := exchange(t, l, Record{Kind: Query, Epoch: 7, ID: 1}, time.Second); reply.Payload[0] != 5 {
		t.Fatal("lost timeout")
	}
	exchange(t, l, Record{Kind: Abort, Epoch: 7, ID: 1}, time.Second)
}

func TestLinkInvalidFramingAndWriterFailure(t *testing.T) {
	l, _ := linkFixture(t)
	if err := l.Tick(0); err != nil {
		t.Fatal(err)
	}
	if err := l.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if err := l.Push(make([]byte, HeaderSize), 0, func([]byte) error { return nil }); err == nil {
		t.Fatal("bad magic")
	}
	var buf [MaxRecord]byte
	n, err := Encode(buf[:], Record{Kind: Hello, Epoch: 7})
	if err != nil {
		t.Fatal(err)
	}
	broken := errors.New("write")
	if err = l.Push(buf[:n], 0, func([]byte) error { return broken }); !errors.Is(err, broken) {
		t.Fatal(err)
	}
	if _, err = NewLink(streamrx.Config{}, new(testSink), time.Second); err == nil {
		t.Fatal("bad config")
	}
	if _, err = NewLink(streamrx.Config{}, new(testSink), 0); err == nil {
		t.Fatal("bad interval")
	}
}

type badIO struct {
	read    io.Reader
	written int
	err     error
	zero    bool
}

func (b *badIO) Read(p []byte) (int, error) {
	if b.zero {
		return 0, nil
	}
	return b.read.Read(p)
}
func (b *badIO) Write(p []byte) (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	if b.zero {
		return 0, nil
	}
	b.written += len(p)
	return len(p), nil
}

func TestClientRejectsBrokenIOAndReply(t *testing.T) {
	for _, transport := range []*badIO{{zero: true}, {err: io.ErrClosedPipe}, {read: bytes.NewReader(nil)}, {read: bytes.NewReader(make([]byte, HeaderSize))}} {
		c := client{stream: transport}
		if _, err := c.exchange(Record{Kind: Hello}); err == nil {
			t.Fatal("bad I/O accepted")
		}
	}
	if _, err := (progressReader{&badIO{zero: true}}).Read(make([]byte, 1)); err != io.ErrNoProgress {
		t.Fatal(err)
	}
	for _, reply := range []Record{{Kind: Hello}, {Kind: Reply, Epoch: 8}, {Kind: Reply, Payload: make([]byte, 20)}} {
		if reply.Kind == Reply && reply.Epoch == 0 {
			reply.Payload[0] = 4
		}
		if _, err := validateReply(reply, Record{}); err == nil {
			t.Fatal("bad reply")
		}
	}
	if err := Send(nil, 1, display.Frame{}); err == nil {
		t.Fatal("nil stream")
	}
	if err := Send(&badIO{}, 1, display.Frame{}); err == nil {
		t.Fatal("empty frame")
	}
}

func TestCodecLimitsAndErrorCodes(t *testing.T) {
	var buf [MaxRecord]byte
	for _, r := range []Record{{Kind: 0}, {Kind: Data, Payload: make([]byte, 101)}} {
		if _, err := Encode(buf[:], r); err == nil {
			t.Fatal("bounds")
		}
	}
	if _, err := Encode(nil, Record{Kind: Hello}); err == nil {
		t.Fatal("short destination")
	}
	for _, err := range []error{ErrRecord, streamrx.ErrState, streamrx.ErrChunk, streamrx.ErrDigest, streamrx.ErrTimeout, errDeferred, io.ErrClosedPipe} {
		if errorCode(err) == 0 {
			t.Fatal("lost error")
		}
	}
}
