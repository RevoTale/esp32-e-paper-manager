package screenclient

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

type recordingSink struct {
	planes  [2][]byte
	commits int
}

func (s *recordingSink) Begin() error { s.planes = [2][]byte{}; return nil }
func (s *recordingSink) Write(pass uint8, _ uint32, p []byte) error {
	s.planes[pass] = append(s.planes[pass], p...)
	return nil
}
func (s *recordingSink) Commit() error { s.commits++; return nil }
func (s *recordingSink) Abort() error  { return nil }

type transport struct {
	connection *screenlink.Connection
	input      bytes.Buffer
	loseCommit bool
	now        time.Duration
	wireBytes  int
	dropKind   screenwire.Kind
	before     bool
	rewrite    func(screenwire.Record, []byte) []byte
}

func (t *transport) Write(p []byte) (int, error) {
	r, err := screenwire.Decode(p)
	if err != nil {
		return 0, err
	}
	if t.before && r.Kind == t.dropKind {
		return 0, io.ErrUnexpectedEOF
	}
	err = t.connection.Push(p, t.now, func(reply []byte) error {
		if t.lost(r.Kind) {
			return io.ErrUnexpectedEOF
		}
		if t.rewrite != nil {
			reply = t.rewrite(r, reply)
		}
		t.wireBytes += len(reply)
		_, err := t.input.Write(reply)
		return err
	})
	if err != nil {
		return 0, err
	}
	t.wireBytes += len(p)
	return len(p), nil
}
func (t *transport) lost(kind screenwire.Kind) bool {
	return (t.loseCommit && kind == screenwire.Commit) || (!t.before && kind == t.dropKind)
}
func (t *transport) Read(p []byte) (int, error) { return t.input.Read(p[:min(len(p), 7)]) }

func fixture(t *testing.T) (*Client, *screenlink.Device, *transport, *recordingSink, display.Frame) {
	t.Helper()
	sink := &recordingSink{}
	caps := screenwire.Capabilities{Width: 17, Height: 9, Stride: 3, MaxChunk: 10, Passes: 2, Format: screenwire.Mono1, Features: screenwire.RawFull, Profile: 1, ProfileVersion: 1, MinimumFullMS: 1000}
	d, err := screenlink.New(caps, [16]byte{1}, sink, time.Second, 2*time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	rw := &transport{connection: d.Open(), now: time.Second}
	c, err := New(bytes.NewReader(bytes.Repeat([]byte{2}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Connect(rw); err != nil {
		t.Fatal(err)
	}
	pixels := bytes.Repeat([]byte{0x80}, 27)
	f, err := display.NewFrame(display.Size{Width: 17, Height: 9}, 3, pixels)
	if err != nil {
		t.Fatal(err)
	}
	return c, d, rw, sink, f
}

func TestNativeClientSendsExactOddWidthPlanes(t *testing.T) {
	c, _, _, sink, f := fixture(t)
	if err := c.Send(f); err != nil {
		t.Fatal(err)
	}
	for _, plane := range sink.planes {
		if !bytes.Equal(plane, f.Bytes()) {
			t.Fatal(plane)
		}
	}
	if sink.commits != 1 || c.Pending() {
		t.Fatal(sink, c.Pending())
	}
}

func TestLostCommitACKReconcilesWithoutNewSPI(t *testing.T) {
	c, d, rw, sink, f := fixture(t)
	rw.loseCommit = true
	if err := c.Send(f); !errors.Is(err, io.ErrUnexpectedEOF) || !c.Pending() {
		t.Fatal(err)
	}
	rw = &transport{connection: d.Open(), now: time.Second}
	if _, err := c.Connect(rw); err != nil {
		t.Fatal(err)
	}
	result, err := c.Reconcile()
	if err != nil || !result.Confirmed || c.Pending() || sink.commits != 1 {
		t.Fatal(result, err, sink)
	}
}

func TestChangedBootDoesNotAdoptHelloLease(t *testing.T) {
	c, d, rw, _, _ := fixture(t)
	if err := rw.connection.Disconnect(); err != nil {
		t.Fatal(err)
	}
	other, err := screenlink.New(c.caps, [16]byte{7}, &recordingSink{}, time.Second, 2*time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Connect(&transport{connection: other.Open(), now: time.Second}); !errors.Is(err, ErrResync) {
		t.Fatal(err)
	}
	if _, err = c.Connect(&transport{connection: d.Open(), now: time.Second}); err != nil {
		t.Fatal(err)
	}
}
